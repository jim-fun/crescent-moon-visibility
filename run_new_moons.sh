#!/bin/bash

# Script to run crescent moon visibility calculations for new moon dates
# Includes first three days after each new moon
# New moon dates calculated using ephem library
#
# Usage: ./run_new_moons.sh [YEAR]
# Example: ./run_new_moons.sh 2027

# Year to process (defaults to current year if not specified)
YEAR=${1:-$(date +%Y)}

echo "Running crescent moon visibility calculations for all new moons in ${YEAR}..."
echo "Processing Day 1, Day 2, and Day 3 for each month"

# Generate new moon dates for the specified year
echo "Calculating new moon dates for ${YEAR}..."

# Use Python to calculate new moon dates
NEW_MOON_DATES=($(python3 << EOF
import datetime
import ephem

year = ${YEAR}
new_moons = []

date = ephem.Date(datetime.date(year, 1, 1))
while date.datetime().year == year:
    date = ephem.next_new_moon(date)
    if date.datetime().year == year:
        new_moons.append(date.datetime().strftime("%Y-%m-%d"))

for date in new_moons:
    print(date)
EOF
))

# Function to add days to a date in YYYY-MM-DD format
add_days() {
    local base_date=$1
    local days=$2
    date -j -v+${days}d -f "%Y-%m-%d" "$base_date" "+%Y-%m-%d" 2>/dev/null || date -d "$base_date + $days day" "+%Y-%m-%d"
}

# Run visibility calculations for each date
for date in "${NEW_MOON_DATES[@]}"; do
    day2=$(add_days "$date" 1)
    day3=$(add_days "$date" 2)

    echo ""
    echo "================================================"
    echo "Processing new moon: $date"
    echo "================================================"

    # Day 1 - New moon date
    echo ""
    echo "--- Day 1: $date ---"
    NOOPEN=1 bash run.sh "$date"

    if [ $? -eq 0 ]; then
        echo "✓ Successfully generated visibility map for $date (Day 1)"
    else
        echo "✗ Failed to generate visibility map for $date (Day 1)"
    fi

    # Day 2 - Day after new moon
    echo ""
    echo "--- Day 2: $day2 ---"
    NOOPEN=1 bash run.sh "$day2"

    if [ $? -eq 0 ]; then
        echo "✓ Successfully generated visibility map for $day2 (Day 2)"
    else
        echo "✗ Failed to generate visibility map for $day2 (Day 2)"
    fi

    # Day 3 - Two days after new moon
    echo ""
    echo "--- Day 3: $day3 ---"
    NOOPEN=1 bash run.sh "$day3"

    if [ $? -eq 0 ]; then
        echo "✓ Successfully generated visibility map for $day3 (Day 3)"
    else
        echo "✗ Failed to generate visibility map for $day3 (Day 3)"
    fi
done

echo ""
echo "================================================"
echo "All new moon visibility calculations completed!"
echo "Generated maps for Day 1, Day 2, and Day 3 for each month"
echo "================================================"
