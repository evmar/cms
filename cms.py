#!/usr/bin/python

import os.path
import markdown
import time
import datetime
import re
import util
import StringIO

import template

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
    try:
        headers, content = util.read_header_file(path)
    except:
        print "when processing '%s':" % path
        raise

    mtime = time.localtime(os.path.getmtime(path))

    attrs = {'content': markdown.markdown(content),
             'lastupdate': time.strftime('%Y-%m-%d', mtime),
             'root': '../' * (path.count('/') - 1)}
    attrs.update(headers)

    output = default_template.evaluate(attrs)

    output_path = os.path.splitext(path)[0] + '.html'
    util.write_if_changed(output_path, output)

default_template = template.Template('site/page.tmpl')
all_files = find_files()
for path in all_files:
    process(default_template, path)
