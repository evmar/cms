#!/usr/bin/python

import os.path

def read_file(path):
    f = open(path)
    data = f.read()
    f.close()
    return data


def write_if_changed(path, output):
    if os.path.exists(path) and read_file(path) == output:
        return
    print '*', path
    with open(path, 'w') as f:
        f.write(output)


def parse_headers(text):
    headers = {}
    for header in text.split('\n'):
        key, val = header.split(': ', 1)
        key = key.lower()
        headers[key] = val
    return headers


def read_header_file(path):
    headertext, content = read_file(path).split('\n\n', 1)
    return parse_headers(headertext), content
