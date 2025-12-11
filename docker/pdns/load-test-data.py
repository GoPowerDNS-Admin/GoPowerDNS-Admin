#!/usr/bin/env python3
"""
Load PowerDNS test data from JSON file.
Alternative to the bash script, doesn't require jq.
"""

import json
import time
import sys
from pathlib import Path
from typing import Dict, Any
import urllib.request
import urllib.error

# Configuration
API_KEY = "my_dev_api_key"
API_URL = "http://localhost:8081/api/v1/servers/localhost"
SCRIPT_DIR = Path(__file__).parent
TEST_DATA_FILE = SCRIPT_DIR / "test-data.json"

# ANSI colors
GREEN = '\033[0;32m'
YELLOW = '\033[1;33m'
RED = '\033[0;31m'
NC = '\033[0m'


def make_request(url: str, method: str = "GET", data: Dict[Any, Any] = None) -> tuple:
    """Make HTTP request to PowerDNS API."""
    headers = {
        "X-API-Key": API_KEY,
        "Content-Type": "application/json"
    }

    request_data = json.dumps(data).encode('utf-8') if data else None
    req = urllib.request.Request(url, data=request_data, headers=headers, method=method)

    try:
        with urllib.request.urlopen(req) as response:
            body = response.read().decode('utf-8')
            return response.status, body if body else None
    except urllib.error.HTTPError as e:
        return e.code, e.read().decode('utf-8')
    except Exception as e:
        return None, str(e)


def wait_for_api(max_attempts: int = 30) -> bool:
    """Wait for PowerDNS API to be ready."""
    print(f"{YELLOW}Waiting for PowerDNS API to be ready...{NC}")

    for _ in range(max_attempts):
        status, _ = make_request(f"{API_URL}/zones")
        if status == 200:
            print(f"{GREEN}PowerDNS API is ready!{NC}\n")
            return True
        print(".", end="", flush=True)
        time.sleep(1)

    print(f"\n{RED}Error: PowerDNS API did not become ready in time{NC}")
    return False


def create_zone(zone_data: Dict[str, Any]) -> bool:
    """Create a zone in PowerDNS."""
    zone_name = zone_data["name"]
    zone_kind = zone_data["kind"]

    print(f"{YELLOW}Creating zone: {NC}{zone_name} ({zone_kind})")

    # Prepare zone creation payload
    if zone_kind == "Slave":
        zone_creation = {
            "name": zone_name,
            "kind": zone_kind,
            "masters": zone_data.get("masters", []),
            "nameservers": zone_data.get("nameservers", [])
        }
    else:
        zone_creation = {k: v for k, v in zone_data.items() if k != "rrsets"}

    # Create the zone
    status, body = make_request(f"{API_URL}/zones", method="POST", data=zone_creation)

    if status == 201:
        print(f"{GREEN}✓ Zone created successfully{NC}")

        # For Native and Master zones, add the records
        if zone_kind != "Slave" and "rrsets" in zone_data and zone_data["rrsets"]:
            rrsets = zone_data["rrsets"]
            print(f"{YELLOW}  Adding {len(rrsets)} record sets...{NC}")

            rrsets_payload = {"rrsets": rrsets}
            patch_status, _ = make_request(
                f"{API_URL}/zones/{zone_name}",
                method="PATCH",
                data=rrsets_payload
            )

            if patch_status in (200, 204):
                print(f"{GREEN}  ✓ Records added successfully{NC}")
            else:
                print(f"{RED}  ✗ Failed to add records (HTTP {patch_status}){NC}")
                return False

        return True
    elif status == 409:
        print(f"{YELLOW}⚠ Zone already exists, skipping{NC}")
        return True
    else:
        print(f"{RED}✗ Failed to create zone (HTTP {status}){NC}")
        try:
            error_data = json.loads(body)
            print(json.dumps(error_data, indent=2))
        except Exception:
            print(body)
        return False


def main():
    """Main function."""
    print(f"{YELLOW}Loading PowerDNS test data...{NC}\n")

    # Check if test data file exists
    if not TEST_DATA_FILE.exists():
        print(f"{RED}Error: Test data file not found at {TEST_DATA_FILE}{NC}")
        sys.exit(1)

    # Wait for API
    if not wait_for_api():
        sys.exit(1)

    # Load test data
    with open(TEST_DATA_FILE, 'r') as f:
        test_data = json.load(f)

    zones = test_data.get("zones", [])
    print(f"Found {GREEN}{len(zones)}{NC} zones to create\n")

    # Create each zone
    success_count = 0
    for zone_data in zones:
        if create_zone(zone_data):
            success_count += 1
        print()

    print(f"{GREEN}Test data loading complete!{NC}")
    print(f"Successfully processed {success_count}/{len(zones)} zones\n")

    # Display summary
    print(f"{YELLOW}Zone Summary:{NC}")
    status, body = make_request(f"{API_URL}/zones")
    if status == 200:
        zones_list = json.loads(body)
        for zone in zones_list:
            print(f"  - {zone['name']} ({zone['kind']})")

    print(f"\n{YELLOW}To query a zone:{NC}")
    print("  dig @localhost example.com")
    print("  dig @localhost www.example.com")

    print(f"\n{YELLOW}To view zones via API:{NC}")
    print(f"  curl -H 'X-API-Key: {API_KEY}' {API_URL}/zones | python3 -m json.tool")

    print(f"\n{YELLOW}To delete all test zones:{NC}")
    print("  Run this script with --delete flag (not implemented yet)")


if __name__ == "__main__":
    try:
        main()
    except KeyboardInterrupt:
        print(f"\n{YELLOW}Interrupted by user{NC}")
        sys.exit(1)
    except SystemExit:
        raise
    except Exception as e:
        print(f"{RED}Error: {e}{NC}")
        sys.exit(1)
