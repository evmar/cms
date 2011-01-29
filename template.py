#!/usr/bin/python

import util

class Template(object):
    def __init__(self, path):
        self.parse(util.read_file(path))

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

