#!/usr/bin/env python3
"""
Unified Code Generation CLI
Main CLI wrapper for generating Go and TypeScript code from OpenAPI specs.
"""

import sys
from pathlib import Path
from typing import Any, Dict, List, Optional


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
sys.path.insert(0, str(REPO_ROOT / "scripts" / "api_generator"))

from generate_go import OGEN_VERSION, GoGenerator
from generate_typescript import TypeScriptGenerator


class GenerationCLI:
    """Unified CLI for code generation."""

    def __init__(self, repo_root: Path = REPO_ROOT):
        self.repo_root = repo_root
        self.go_gen = GoGenerator(repo_root)
        self.ts_gen = TypeScriptGenerator(repo_root)

    def display_menu(self, services: List[Dict]) -> None:
        """Display the main menu."""
        print("\n" + "=" * 60)
        print("OpenAPI Code Generation CLI")
        print(f"Go: ogen {OGEN_VERSION} | TypeScript: openapi-typescript")
        print("=" * 60)

        print("\nAvailable Services:")
        for idx, service in enumerate(services, 1):
            print(f"{idx:2d}. {service['name']}")

        print(f"\nGeneration Options:")
        print(f" 0. Generate ALL services (Go + TypeScript)")
        print(f" g. Generate Go code only (all services)")
        print(f" t. Generate TypeScript bindings only (all services)")
        print(f" q. Quit")
        print("=" * 60)

    def get_user_choice(self, services: List[Dict]) -> str:
        """Get user's selection."""
        while True:
            choice = input("\nSelect an option (number, g, t, or q): ").strip().lower()

            if choice in ["q", "g", "t", "0"]:
                return choice

            try:
                idx = int(choice)
                if 1 <= idx <= len(services):
                    return choice
                else:
                    print(
                        f"Invalid choice. Please enter a number between 1 and {len(services)}"
                    )
            except ValueError:
                print("Invalid input. Please enter a number, 'g', 't', or 'q'.")

    def generate_single_service(
        self, service: Dict, generate_go: bool, generate_ts: bool
    ) -> bool:
        """Generate code for a single service."""
        service_name = service["name"]
        success = True

        print(f"\n{'='*60}")
        print(f"Generating for: {service_name}")
        print(f"{'='*60}")

        if generate_go:
            print("\n🔨 Generating Go code...")
            if not self.go_gen.generate_for_service(service, verbose=True):
                success = False

        if generate_ts:
            print("\n📘 Generating TypeScript bindings...")
            ts_service = None
            for ts_svc in self.ts_gen.find_services_with_openapi():
                if ts_svc["name"] == service_name:
                    ts_service = ts_svc
                    break

            if ts_service:
                if not self.ts_gen.generate_for_service(ts_service, verbose=True):
                    success = False
                # Regenerate barrel export
                all_ts_services = self.ts_gen.find_services_with_openapi()
                self.ts_gen.generate_barrel_export(all_ts_services)
            else:
                print(f"⚠️  No TypeScript spec found for {service_name}")

        return success

    def generate_all_services(self, generate_go: bool, generate_ts: bool) -> None:
        """Generate code for all services."""
        results: Dict[str, Optional[Dict[str, Any]]] = {"go": None, "ts": None}

        print(f"\n{'='*60}")
        print("Generating for ALL services")
        print(f"{'='*60}\n")

        if generate_go:
            print("🔨 Generating Go code...\n")
            results["go"] = self.go_gen.generate_for_all(verbose=True)

        if generate_ts:
            print("\n📘 Generating TypeScript bindings...\n")
            results["ts"] = self.ts_gen.generate_for_all(verbose=True)

        # Print summary
        print(f"\n{'='*60}")
        print("Generation Summary")
        print(f"{'='*60}")

        if results["go"]:
            print(
                f"\nGo: {results['go']['success']}/{results['go']['total']} services succeeded"
            )
            if results["go"]["failed_services"]:
                print(f"  Failed: {', '.join(results['go']['failed_services'])}")

        if results["ts"]:
            print(
                f"\nTypeScript: {results['ts']['success']}/{results['ts']['total']} services succeeded"
            )
            if results["ts"]["failed_services"]:
                print(f"  Failed: {', '.join(results['ts']['failed_services'])}")

        print(f"\n{'='*60}\n")

    def run(self) -> int:
        """Run the CLI."""
        print(f"\nOpenAPI Code Generation CLI")
        print(f"Repository: {self.repo_root.name}")

        # Get services
        try:
            services = self.go_gen.find_services_with_openapi()
        except FileNotFoundError as e:
            print(f"\n❌ Error: {e}")
            return 1

        if not services:
            print("\n❌ No services with OpenAPI specs found!")
            return 1

        # Display menu and get choice
        self.display_menu(services)
        choice = self.get_user_choice(services)

        # Process choice
        if choice == "q":
            print("\nExiting...")
            return 0

        elif choice == "0":
            # Generate both for all services
            self.generate_all_services(generate_go=True, generate_ts=True)

        elif choice == "g":
            # Generate Go only for all services
            self.generate_all_services(generate_go=True, generate_ts=False)

        elif choice == "t":
            # Generate TypeScript only for all services
            self.generate_all_services(generate_go=False, generate_ts=True)

        else:
            # Single service
            idx = int(choice) - 1
            service = services[idx]

            # Ask what to generate
            print(f"\nSelected: {service['name']}")
            print("What would you like to generate?")
            print("1. Go only")
            print("2. TypeScript only")
            print("3. Both")

            gen_choice = input("\nChoice (1-3): ").strip()

            generate_go = gen_choice in ["1", "3"]
            generate_ts = gen_choice in ["2", "3"]

            self.generate_single_service(service, generate_go, generate_ts)

        return 0


def main():
    """Main entry point."""
    cli = GenerationCLI()
    try:
        return cli.run()
    except KeyboardInterrupt:
        print("\n\nInterrupted by user. Exiting...")
        return 130
    except Exception as e:
        print(f"\n❌ Fatal error: {e}")
        import traceback

        traceback.print_exc()
        return 1


if __name__ == "__main__":
    sys.exit(main())
