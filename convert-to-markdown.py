#!/usr/bin/python

import re
import sys
import textwrap

with open(sys.argv[1]) as f:
    content = f.read()

blocks = content.strip().split('\n\n')
headers, blocks = blocks[0], blocks[1:]

for header in headers.split('\n'):
    print header[1:]  # remove % prefix
print

allblocks = []
for block in blocks:
    lines = block.split('\n')
    while lines:
        line = lines[0]
        if line.startswith('== '):
            allblocks.append(lines.pop(0))
        elif line.startswith('- '):
            block = lines.pop(0)
            while lines and lines[0].startswith('  '):
                block += ' ' + lines.pop(0).strip()
            allblocks.append(block)
        else:
            block = ''
            while (line and not line.startswith('== ')
                   and not line.startswith('- ')):
                block += ' ' + lines.pop(0).strip()
                if lines:
                    line = lines[0]
                else:
                    line = None
            allblocks.append(block.strip())

def wrap(text):
    return textwrap.fill(text, width=72)

for block in allblocks:
    def linkrepl(match):
        text, target = match.groups()
        return '[%s](%s)' % (text, target)
    block = re.sub(r'\[\[(.+?) {(.+?)}\]\]', linkrepl, block)
    if block.startswith('== '):
        print wrap('## ' + block[3:])
    elif block.startswith('- '):
        print wrap('* ' + block[2:])
    else:
        print wrap(block)
    print
