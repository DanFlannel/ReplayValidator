#!/usr/bin/env python3
"""
TypeScript OpenAPI Client Generator
Generates TypeScript client bindings from OpenAPI specifications.
"""

import re
import subprocess
from pathlib import Path
from typing import Any, Dict, List


# Repo root resolution - find the git root
def find_repo_root() -> Path:
    """Find the repository root by looking for .git directory."""
    current = Path(__file__).resolve()
    for parent in [current] + list(current.parents):
        if (parent / ".git").exists():
            return parent
    # Fallback: assume scripts/api_generator structure
    return Path(__file__).parent.parent.parent.resolve()


REPO_ROOT = find_repo_root()


class TypeScriptGenerator:
    """Handles TypeScript client generation using openapi-typescript."""

    def __init__(self, repo_root: Path):
        self.repo_root = repo_root
        self.services_dir = repo_root / "services"
        self.packages_api_dir = repo_root / "packages" / "api"
        self.output_dir = self.packages_api_dir / "src" / "generated"

    def find_services_with_openapi(self) -> List[Dict[str, Any]]:
        """Find all services that have an openapi spec."""
        services = []

        if not self.services_dir.exists():
            raise FileNotFoundError(
                f"Services directory not found at {self.services_dir}"
            )

        # Search for openapi specs in services/*/openapi/ folders
        for service_dir in sorted(self.services_dir.iterdir()):
            if service_dir.is_dir():
                openapi_dir = service_dir / "openapi"

                # Check for yaml, yml, or json
                for ext in ["yaml", "yml", "json"]:
                    spec_path = openapi_dir / f"openapi.{ext}"
                    if spec_path.exists():
                        services.append(
                            {
                                "name": service_dir.name,  # str
                                "spec_path": spec_path,  # Path
                                "output_path": self.output_dir
                                / service_dir.name,  # Path
                            }
                        )
                        break  # Only take first matching spec

        return services

    def to_pascal_case(self, s: str) -> str:
        """Convert string to PascalCase (e.g., 'search-engine' -> 'SearchEngine')."""
        parts = re.split(r"[-_]", s)
        return "".join(word.capitalize() for word in parts)

    def sanitize_name(self, s: str) -> str:
        """Sanitize service name for use in identifiers (e.g., 'search-engine' -> 'SEARCH_ENGINE')."""
        return s.upper().replace("-", "_")

    def create_client_wrapper(self, service_name: str, output_path: Path) -> None:
        """Create TypeScript client wrapper file."""
        wrapper_path = output_path / "index.ts"
        pascal_name = self.to_pascal_case(service_name)
        env_var = self.sanitize_name(service_name)

        content = f"""import createClient from 'openapi-fetch';
import type {{ paths }} from './schema';

export type {pascal_name}Paths = paths;

export interface {pascal_name}ClientConfig {{
  baseUrl?: string;
  headers?: HeadersInit;
  fetch?: typeof fetch;
}}

/**
 * Create a {service_name} API client
 *
 * @param config - Client configuration
 * @param config.baseUrl - Base URL for the API (overrides environment variables)
 *
 * @example
 * // Use environment variable or default
 * const client = create{pascal_name}Client();
 *
 * // Override for local Docker development
 * const client = create{pascal_name}Client({{
 *   baseUrl: 'http://localhost:3001'
 * }});
 *
 * // With custom headers
 * const client = create{pascal_name}Client({{
 *   baseUrl: 'http://localhost:3001',
 *   headers: {{ 'X-Custom-Header': 'value' }}
 * }});
 */
export function create{pascal_name}Client(config?: {pascal_name}ClientConfig) {{
  // Try multiple environment variable patterns for cross-platform support
  const baseUrl = config?.baseUrl
    || process.env.{env_var}_API_URL
    || process.env.VITE_{env_var}_API_URL  // Vite (web/desktop)
    || process.env.NEXT_PUBLIC_{env_var}_API_URL  // Next.js
    || process.env.EXPO_PUBLIC_{env_var}_API_URL  // Expo (mobile)
    || 'http://localhost:8080';  // Development fallback

  return createClient<paths>({{
    baseUrl,
    headers: config?.headers,
    fetch: config?.fetch,
  }});
}}

// Re-export types for convenience
export type * from './schema';
"""

        wrapper_path.write_text(content)

    def generate_barrel_export(self, services: List[Dict[str, Any]]) -> None:
        """Generate barrel export file."""
        index_path = self.output_dir / "index.ts"

        exports = []
        for service in services:
            pascal_name = self.to_pascal_case(service["name"])
            exports.append(
                f"export {{ create{pascal_name}Client, type {pascal_name}Paths }} from './{service['name']}';"
            )

        content = f"""// Auto-generated barrel export
// Do not edit this file manually

{chr(10).join(exports)}
"""

        index_path.write_text(content)
        print(
            f"📝 Generated barrel export at {index_path.relative_to(self.repo_root)}\n"
        )

    def generate_for_service(
        self, service: Dict[str, Any], verbose: bool = False
    ) -> bool:
        """Generate TypeScript client for a specific service."""
        service_name: str = service["name"]
        spec_path: Path = service["spec_path"]
        output_path: Path = service["output_path"]

        if verbose:
            print(f"📦 Generating TypeScript client for: {service_name}")
            print(f"   Spec: {spec_path}")
            print(f"   Output: {output_path}\n")

        try:
            # Ensure output directory exists
            output_path.mkdir(parents=True, exist_ok=True)

            schema_path = output_path / "schema.ts"

            # Generate TypeScript types from OpenAPI spec
            cmd = ["npx", "openapi-typescript", str(spec_path), "-o", str(schema_path)]

            if verbose:
                print(f"   Running: {' '.join(cmd)}")

            result = subprocess.run(
                cmd,
                cwd=self.packages_api_dir,
                check=True,
                capture_output=not verbose,
                text=True,
            )

            if verbose and result.stdout:
                print(result.stdout)

            # Create client wrapper
            self.create_client_wrapper(service_name, output_path)

            if verbose:
                print(f"   ✅ Generated successfully\n")

            return True

        except subprocess.CalledProcessError as e:
            error_msg = e.stderr if e.stderr else str(e)
            print(f"   ❌ Failed to generate client for {service_name}: {error_msg}")
            print()
            return False
        except Exception as e:
            print(f"   ❌ Unexpected error for {service_name}: {e}")
            print()
            return False

    def generate_for_all(self, verbose: bool = False) -> Dict[str, Any]:
        """Generate TypeScript clients for all services."""
        # Ensure output directory exists
        self.output_dir.mkdir(parents=True, exist_ok=True)

        services = self.find_services_with_openapi()

        if not services:
            return {"success": 0, "failed": 0, "total": 0, "failed_services": []}

        if verbose:
            print(f"✨ Found {len(services)} OpenAPI spec(s)\n")

        success_count = 0
        failed_services = []

        for service in services:
            if self.generate_for_service(service, verbose=verbose):
                success_count += 1
            else:
                failed_services.append(service["name"])

        # Generate barrel export
        if services:
            self.generate_barrel_export(services)

        return {
            "success": success_count,
            "failed": len(failed_services),
            "total": len(services),
            "failed_services": failed_services,
        }


def main():
    """Standalone execution."""
    generator = TypeScriptGenerator(REPO_ROOT)

    print("\n🔍 TypeScript OpenAPI Client Generator\n")

    services = generator.find_services_with_openapi()

    if not services:
        print("⚠️  No OpenAPI specs found in services directory")
        return 0

    print(f"Found {len(services)} services with OpenAPI specs")
    print("Generating TypeScript clients...\n")

    result = generator.generate_for_all(verbose=True)

    print(f"{'='*60}")
    print(f"Summary: {result['success']}/{result['total']} services succeeded")
    if result["failed_services"]:
        print(f"Failed services: {', '.join(result['failed_services'])}")
    print(f"{'='*60}\n")

    print(f"📁 Generated files location: {generator.output_dir}")

    return 0 if result["failed"] == 0 else 1


if __name__ == "__main__":
    import sys

    sys.exit(main())
