#!/usr/bin/env python3
import datetime
import sys
import ephem

def get_new_moons_in_year(year: int):
    """Returns a list of new moon dates in a year"""
    new_moons = []

    date = ephem.Date(datetime.date(year, 1, 1))
    while date.datetime().year == year:
        date = ephem.next_new_moon(date)
        if date.datetime().year == year:
            new_moons.append(date)

    return new_moons

if __name__ == "__main__":
    # Accept year as command-line argument, default to current year
    year = int(sys.argv[1]) if len(sys.argv) > 1 else datetime.datetime.now().year
    new_moons = get_new_moons_in_year(year)

    print(f"New Moon dates in {year}:")
    for moon_date in new_moons:
        # Convert to datetime and format as YYYY-MM-DD
        dt = moon_date.datetime()
        print(dt.strftime("%Y-%m-%d"))
