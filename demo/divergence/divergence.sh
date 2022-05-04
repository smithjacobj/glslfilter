#!/bin/bash

if [ -z $1 ]
then
  echo "Usage: divergence.sh <input-png-path>"
  exit 1
fi

script_path="$( dirname -- ${BASH_SOURCE} )"
executable_path="${script_path}/../../glslfilter-glfw/glslfilter-glfw"
filter_path="${script_path}/divergence.yml"

cygpath_exists=! cygpath &> /dev/null

if ($cygpath_exists)
then
  set - $( cygpath -w "${1}" | sed 's%\\%/%g' )
fi

identify_exists=! identify --version &> /dev/null

if ! $identify_exists
then
  echo "identify command requires imagemagick package"
  exit 1
fi

input_width=$( identify -format '%[fx:w]' ${1} )
input_height=$( identify -format '%[fx:h]' ${1} )
let output_width=${input_width}*8
let output_height=${input_height}*8

sed "s%\${source_path}%${1}%g" $filter_path |
sed "s%\${script_path}%${script_path}%g" |
sed "s%\${width}%${output_width}%g" |
sed "s%\${height}%${output_height}%g" |
$executable_path
