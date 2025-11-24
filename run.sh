#!/bin/bash

CC=gcc
OPEN=xdg-open
if [[ "$OSTYPE" == "darwin"* ]]; then
    CC="/opt/homebrew/opt/llvm/bin/clang -L/opt/homebrew/opt/llvm/lib -fno-rtti" # -mllvm -polly
    OPEN=open
fi

rm -f visibility.out

# Resolution settings: 4=HD, 8=Full HD, 16=4K
RESOLUTION=${RESOLUTION:-16}
$CC -fopenmp -O3 -Wall -Werror -o visibility.out -fno-exceptions -DPIXEL_PER_DEGREE=$RESOLUTION visibility.cc thirdparty/astronomy.c -lm || exit $?

echo "Compiliation is completed (${RESOLUTION}x resolution), now let's run the code."

DATE=$1 #YYYY-MM-DD
TYPE=evening
METHOD=yallop
time ./visibility.out $DATE map $TYPE $METHOD $DATE.png || (echo Not successful && exit 1)
composite -blend 60 $DATE.png map.png $DATE.png
TYPE="$(tr '[:lower:]' '[:upper:]' <<<${TYPE:0:1})${TYPE:1}"
METHOD="$(tr '[:lower:]' '[:upper:]' <<<${METHOD:0:1})${METHOD:1}"

# Use magick instead of convert to avoid ImageMagick v7 deprecation warning
# Scale font size based on resolution (base 20 for PIXEL_PER_DEGREE=4)
FONTSIZE=$((20 * RESOLUTION / 4))
magick $DATE.png -pointsize $FONTSIZE -fill black -draw "gravity south text 0,10 '$TYPE, $METHOD, $DATE'" $DATE.png

# Only open the image if NOOPEN is not set
if [ -z "$NOOPEN" ]; then
    $OPEN $DATE.png
fi
