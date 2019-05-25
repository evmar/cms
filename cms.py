#!/usr/bin/python

import os.path
import markdown
import time
import datetime
import re
import util
import StringIO

import jinja2

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


def process(jinja, path):
    try:
        headers, content = util.read_header_file(path)
    except:
        print "when processing '%s':" % path
        raise

    html = markdown.markdown(content.decode('utf-8'),
                             extensions=['smarty'],
                             extension_configs={
                                 'smarty': {
                                     'smart_quotes': False,
                                     'smart_ellipses': False,
                                     'substitutions': {
                                         'ndash': '&mdash;',
                                     },
                                 }
                             }).encode('utf-8')
    attrs = {
        'content': html,
        'root': '../' * (path.count('/') - 1),
    }
    attrs.update(headers)

    output = jinja.get_template('page.tmpl').render(
        **attrs
        )

    output_path = os.path.splitext(path)[0] + '.html'
    util.write_if_changed(output_path, output)

all_files = find_files()
jinja = jinja2.Environment(
    loader=jinja2.FileSystemLoader('site'),
    autoescape=True)
for path in all_files:
    process(jinja, path)
