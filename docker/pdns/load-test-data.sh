#!/bin/bash
set -e

# PowerDNS API Configuration
API_KEY="my_dev_api_key"
API_URL="http://localhost:8081/api/v1/servers/localhost"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TEST_DATA_FILE="$SCRIPT_DIR/test-data.json"

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

echo -e "${YELLOW}Loading PowerDNS test data...${NC}\n"

# Check if jq is installed
if ! command -v jq &> /dev/null; then
    echo -e "${RED}Error: jq is required but not installed.${NC}"
    echo "Install it with: brew install jq (macOS) or apt-get install jq (Linux)"
    exit 1
fi

# Check if test data file exists
if [ ! -f "$TEST_DATA_FILE" ]; then
    echo -e "${RED}Error: Test data file not found at $TEST_DATA_FILE${NC}"
    exit 1
fi

# Wait for PowerDNS to be ready
echo -e "${YELLOW}Waiting for PowerDNS API to be ready...${NC}"
max_attempts=30
attempt=0
while [ $attempt -lt $max_attempts ]; do
    if curl -s -H "X-API-Key: $API_KEY" "$API_URL/zones" > /dev/null 2>&1; then
        echo -e "${GREEN}PowerDNS API is ready!${NC}\n"
        break
    fi
    attempt=$((attempt + 1))
    echo -n "."
    sleep 1
done

if [ $attempt -eq $max_attempts ]; then
    echo -e "\n${RED}Error: PowerDNS API did not become ready in time${NC}"
    exit 1
fi

# Get the number of zones
zone_count=$(jq '.zones | length' "$TEST_DATA_FILE")
echo -e "Found ${GREEN}$zone_count${NC} zones to create\n"

# Process each zone
for i in $(seq 0 $((zone_count - 1))); do
    zone_name=$(jq -r ".zones[$i].name" "$TEST_DATA_FILE")
    zone_kind=$(jq -r ".zones[$i].kind" "$TEST_DATA_FILE")

    echo -e "${YELLOW}Creating zone: ${NC}$zone_name (${zone_kind})"

    # Extract zone data
    zone_data=$(jq ".zones[$i]" "$TEST_DATA_FILE")

    # For Slave zones, use a different structure
    if [ "$zone_kind" = "Slave" ]; then
        zone_creation=$(jq -n \
            --arg name "$zone_name" \
            --arg kind "$zone_kind" \
            --argjson masters "$(echo "$zone_data" | jq '.masters')" \
            --argjson nameservers "$(echo "$zone_data" | jq '.nameservers')" \
            '{name: $name, kind: $kind, masters: $masters, nameservers: $nameservers}')
    else
        # For Native and Master zones, include rrsets
        zone_creation=$(echo "$zone_data" | jq 'del(.rrsets)')
    fi

    # Create the zone
    response=$(curl -s -w "\n%{http_code}" -X POST "$API_URL/zones" \
        -H "X-API-Key: $API_KEY" \
        -H "Content-Type: application/json" \
        -d "$zone_creation")

    http_code=$(echo "$response" | tail -n1)
    body=$(echo "$response" | sed '$d')

    if [ "$http_code" = "201" ]; then
        echo -e "${GREEN}✓ Zone created successfully${NC}"

        # For Native and Master zones, add the records
        if [ "$zone_kind" != "Slave" ]; then
            rrsets=$(echo "$zone_data" | jq '.rrsets')
            if [ "$rrsets" != "null" ] && [ "$(echo "$rrsets" | jq 'length')" -gt 0 ]; then
                echo -e "${YELLOW}  Adding $(echo "$rrsets" | jq 'length') record sets...${NC}"

                # Create the rrsets payload
                rrsets_payload=$(jq -n --argjson rrsets "$rrsets" '{rrsets: $rrsets}')

                # Patch the zone with records
                patch_response=$(curl -s -w "\n%{http_code}" -X PATCH "$API_URL/zones/$zone_name" \
                    -H "X-API-Key: $API_KEY" \
                    -H "Content-Type: application/json" \
                    -d "$rrsets_payload")

                patch_http_code=$(echo "$patch_response" | tail -n1)

                if [ "$patch_http_code" = "204" ] || [ "$patch_http_code" = "200" ]; then
                    echo -e "${GREEN}  ✓ Records added successfully${NC}"
                else
                    echo -e "${RED}  ✗ Failed to add records (HTTP $patch_http_code)${NC}"
                    echo "$patch_response" | sed '$d' | jq '.' 2>/dev/null || echo "$patch_response" | sed '$d'
                fi
            fi
        fi
    elif [ "$http_code" = "409" ]; then
        echo -e "${YELLOW}⚠ Zone already exists, skipping${NC}"
    else
        echo -e "${RED}✗ Failed to create zone (HTTP $http_code)${NC}"
        echo "$body" | jq '.' 2>/dev/null || echo "$body"
    fi

    echo ""
done

echo -e "${GREEN}Test data loading complete!${NC}\n"

# Display summary
echo -e "${YELLOW}Zone Summary:${NC}"
curl -s -H "X-API-Key: $API_KEY" "$API_URL/zones" | jq -r '.[] | "  - \(.name) (\(.kind))"'

echo -e "\n${YELLOW}To query a zone:${NC}"
echo "  dig @localhost example.com"
echo "  dig @localhost www.example.com"

echo -e "\n${YELLOW}To view zones via API:${NC}"
echo "  curl -H 'X-API-Key: $API_KEY' $API_URL/zones | jq"

echo -e "\n${YELLOW}To delete all test zones:${NC}"
echo "  curl -H 'X-API-Key: $API_KEY' $API_URL/zones | jq -r '.[].id' | xargs -I {} curl -X DELETE -H 'X-API-Key: $API_KEY' $API_URL/zones/{}"
