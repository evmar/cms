#!/usr/bin/python

import itertools
import datetime
import markdown
import os
import sys

import util
import template

class Post(object):
    def __init__(self, title, timestamp, path, content):
        self.title = title
        self.timestamp = timestamp
        self.path = path
        self.content = content

settings = None
templates = {}

def load_post(path):
    headers, content = util.read_header_file(path)

    timestamp = datetime.datetime.strptime(headers['timestamp'],
                                           '%Y/%m/%d %H:%M')
    #timestamp += datetime.timedelta(seconds=time.timezone)
    html = markdown.markdown(content.decode('utf-8')).encode('utf-8')

    path = (timestamp.strftime('%Y/%m/%d') + '/' +
            os.path.splitext(os.path.basename(path))[0] + '.html')
    return Post(title=headers['subject'],
                timestamp=timestamp,
                path=path,
                content=html)


def load_posts(root):
    posts = []
    def visit(arg, dirname, files):
        for file in files:
            _, ext = os.path.splitext(file)
            if ext == '.text':
                path = os.path.join(dirname, file)
                posts.append(load_post(path))
    os.path.walk(root, visit, None)
    posts.sort(key=lambda post: post.timestamp)
    posts.reverse()
    return posts


def render_post(root, post):
    return templates['post'].evaluate({
            'title': post.title,
            'datetime': post.timestamp.strftime('%Y-%m-%d %H:%M'),
            'url': os.path.join(root, post.path),
            'content': post.content
            })


def generate_archive(posts):
    content = ''
    def month_key(post):
        return post.timestamp.strftime('%Y/%m')
    for month, mposts in itertools.groupby(posts, key=month_key):
        links = ''
        for post in mposts:
            links += templates['archive-post'].evaluate({
                    'url': post.path,
                    'title': post.title
                    })
        content += templates['archive-month'].evaluate({
                'month': month,
                'posts': links
                })
    return templates['page'].evaluate({
                'title': settings['title'] + ': all posted entries',
                'root': './',
                'content': content
                })


def generate_post_page(root, post):
    content = templates['single-post'].evaluate({
            'root': root,
            'post': render_post(root, post)
            })
    return templates['page'].evaluate({
            'title': settings['title'] + ': ' + post.title,
            'root': root,
            'content': content
            })


def generate_index(posts):
    posts = posts[:10]
    return templates['page'].evaluate({
            'title': settings['title'],
            'extrahead': settings['index_extra_head'],
            'content': ''.join([render_post('', p) for p in posts])
            })


def regenerate(in_dir):
    os.umask(022)

    global settings
    settings = util.parse_headers(util.read_file(
            os.path.join(in_dir, 'settings')).strip())

    global templates
    for filename in os.listdir(os.path.join(in_dir, 'templates')):
        name, ext = os.path.splitext(filename)
        if ext == '.tmpl':
            path = os.path.join(in_dir, 'templates', filename)
            templates[name] = template.Template(path)

    posts = load_posts(os.path.join(in_dir, 'posts'))

    util.write_if_changed('index.html', generate_index(posts))
    util.write_if_changed('archive.html', generate_archive(posts))
    for post in posts:
        dir = os.path.split(post.path)[0]
        try:
            os.makedirs(dir)
        except OSError:
            pass
        root = '../' * post.path.count('/')
        util.write_if_changed(post.path, generate_post_page(root, post))

if __name__ == '__main__':
    regenerate(sys.argv[1])
