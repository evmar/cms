#!/usr/bin/python

import os.path
import markdown
import time
import datetime
import re
import sitefeed
import StringIO

def readfile(path):
    f = open(path)
    data = f.read()
    f.close()
    return data


class Template(object):
    def __init__(self, path):
        self.parse(readfile(path))

    def parse(self, text):
        self.parts = text.split('%%')

    def evaluate(self, attrs):
        out = []
        for i, part in enumerate(self.parts):
            if i % 2 == 0:
                out.append(part)
            else:
                out.append(attrs.get(part, ''))
        return ''.join(out)


def find_files():
    all_files = []
    skip_dirs = ('.git', '_darcs')
    def visit(arg, dirname, files):
        files[:] = filter(lambda f: f not in skip_dirs, files)
        for file in files:
            _, ext = os.path.splitext(file)
            if ext == '.cms':
                all_files.append(os.path.join(dirname, file))
    os.path.walk('.', visit, None)
    return all_files


def process(default_template, path):
    headertext, content = readfile(path).split('\n\n', 1)
    headers = {}
    for header in headertext.split('\n'):
        key, val = header.split(': ', 1)
        key = key.lower()
        headers[key] = val

    mtime = time.localtime(os.path.getmtime(path))

    def special(cmd):
        if cmd == 'sitefeed':
            posts = sitefeed.load()
            output = StringIO.StringIO()
            sitefeed.write_html(posts, output)
            return output.getvalue()
        else:
            raise RuntimeError, repr(cmd)

    content = re.sub(r'\n\n%(\w+\S+)\n\n',
                     lambda match: '\n\n' + special(match.group(1)) + '\n\n',
                     content)

    attrs = {'content': markdown.markdown(content),
             'lastupdate': time.strftime('%Y-%m-%d', mtime)}
    attrs.update(headers)

    output = default_template.evaluate(attrs)

    output_path = os.path.splitext(path)[0] + '.html'
    if readfile(output_path) != output:
        print '*', output_path
        with open(output_path, 'w') as f:
            f.write(output)

default_template = Template('site/page.tmpl')
all_files = find_files()
for path in all_files:
    process(default_template, path)
