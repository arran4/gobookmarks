#!/usr/bin/env bash
shopt -s nullglob

for f in *.mp4; do
  name="${f%.mp4}"
  echo "Converting \"$f\" → \"$name.gif\"…"

  # 1) generate palette from the video
  ffmpeg -v warning -i "$f" \
    -vf "fps=10,scale=320:-1:flags=lanczos,palettegen" \
    -y "${name}_palette.png"

  # 2) use that palette to create the final GIF
  ffmpeg -v warning -i "$f" -i "${name}_palette.png" \
    -filter_complex "fps=10,scale=320:-1:flags=lanczos[x];[x][1:v]paletteuse" \
    -loop 0 \
    -y "${name}.gif"

  # clean up
  rm "${name}_palette.png"
done