#!/usr/bin/env python3
"""
Go OpenAPI Code Generator (ogen)
Generates Go server code from OpenAPI specifications.
"""

import re
import shutil
import subprocess
from pathlib import Path
from typing import Any, Dict, List

def _ogen_version_from_go_mod() -> str:
    """Report the ogen version go.mod requires, for display only."""
    go_mod = Path(__file__).resolve().parents[2] / "go.mod"
    match = re.search(r"github\.com/ogen-go/ogen (v\S+)", go_mod.read_text())

    return match.group(1) if match else "unknown"

# Configuration
#
# The generator version is NOT pinned here. It is whatever go.mod requires, and
# the command below runs it without an @version so the two cannot disagree.
# This constant read "v1.8.0" while go.mod required v1.18.0 and the service
# Makefiles ran @latest -- three versions for one generator (A334).
OGEN_VERSION = _ogen_version_from_go_mod()


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


class GoGenerator:
    """Handles Go code generation using ogen."""

    def __init__(self, repo_root: Path, ogen_version: str = OGEN_VERSION):
        self.repo_root = repo_root
        self.services_dir = repo_root / "services"
        self.ogen_version = ogen_version

    def find_services_with_openapi(self) -> List[Dict[str, Any]]:
        """Find all services that have an openapi directory."""
        services = []

        if not self.services_dir.exists():
            raise FileNotFoundError(
                f"Services directory not found at {self.services_dir}"
            )

        for service_dir in sorted(self.services_dir.iterdir()):
            if service_dir.is_dir():
                openapi_dir = service_dir / "openapi"
                openapi_yaml = openapi_dir / "openapi.yaml"

                if openapi_yaml.exists():
                    services.append(
                        {
                            "name": service_dir.name,  # str
                            "path": service_dir,  # Path
                            "openapi_yaml": openapi_yaml,  # Path
                            "openapi_dir": openapi_dir,  # Path
                        }
                    )

        return services

    def generate_for_service(
        self, service: Dict[str, Any], verbose: bool = False
    ) -> bool:
        """Generate Go code for a specific service."""
        service_name: str = service["name"]
        service_path: Path = service["path"]
        openapi_yaml: Path = service["openapi_yaml"]

        if verbose:
            print(f"\n{'='*60}")
            print(f"Generating Go code for: {service_name}")
            print(f"{'='*60}")

        # Command to run ogen
        ogen_cmd = [
            "go",
            "run",
            "github.com/ogen-go/ogen/cmd/ogen",
            "--target",
            "openapi",
            "--clean",
            "./openapi/openapi.yaml",
        ]

        try:
            if verbose:
                print(f"Running: {' '.join(ogen_cmd)}")
                print(f"Working directory: {service_path}\n")

            result = subprocess.run(
                ogen_cmd,
                cwd=service_path,
                check=True,
                capture_output=not verbose,
                text=True,
            )

            if verbose and result.stdout:
                print(result.stdout)

            # Copy openapi.yaml to static folder
            static_dir = service_path / "static"
            static_yaml = static_dir / "openapi.yaml"

            if not static_dir.exists():
                static_dir.mkdir(parents=True)
                if verbose:
                    print(f"Created static directory: {static_dir}")

            # Copy the file
            shutil.copy2(openapi_yaml, static_yaml)

            if verbose:
                print(
                    f"✓ Copied {openapi_yaml.name} to {static_yaml.relative_to(self.repo_root)}"
                )
                print(f"✓ Successfully generated Go code for {service_name}\n")

            return True

        except subprocess.CalledProcessError as e:
            error_msg = e.stderr if e.stderr else str(e)
            print(f"✗ Error generating Go code for {service_name}:")
            print(f"  {error_msg}")
            return False
        except Exception as e:
            print(f"✗ Unexpected error for {service_name}: {e}")
            return False

    def generate_for_all(self, verbose: bool = False) -> Dict[str, Any]:
        """Generate Go code for all services."""
        services = self.find_services_with_openapi()

        if not services:
            return {"success": 0, "failed": 0, "total": 0, "failed_services": []}

        success_count = 0
        failed_services = []

        for service in services:
            if self.generate_for_service(service, verbose=verbose):
                success_count += 1
            else:
                failed_services.append(service["name"])

        return {
            "success": success_count,
            "failed": len(failed_services),
            "total": len(services),
            "failed_services": failed_services,
        }


def main():
    """Standalone execution."""
    generator = GoGenerator(REPO_ROOT)

    print(f"\nGo OpenAPI Code Generator (ogen {OGEN_VERSION})\n")

    services = generator.find_services_with_openapi()

    if not services:
        print("No services with OpenAPI specs found!")
        return 1

    print(f"Found {len(services)} services with OpenAPI specs")
    print("Generating Go code for all services...\n")

    result = generator.generate_for_all(verbose=True)

    print(f"\n{'='*60}")
    print(f"Summary: {result['success']}/{result['total']} services succeeded")
    if result["failed_services"]:
        print(f"Failed services: {', '.join(result['failed_services'])}")
    print(f"{'='*60}\n")

    return 0 if result["failed"] == 0 else 1


if __name__ == "__main__":
    import sys

    sys.exit(main())
