#!/usr/bin/python2

import itertools
import datetime
import markdown
import os
import sys

import jinja2

import atom
import util
import time

class Post(object):
    def __init__(self, title, timestamp, path, content, summary):
        self.title = title
        self.timestamp = timestamp
        self.path = path
        self.content = content
        self.summary = summary

settings = None
jinja = None

FRONTPAGE_POSTS = 10

def load_post(path):
    headers, content = util.read_header_file(path)

    timestamp = datetime.datetime.strptime(headers['timestamp'],
                                           '%Y/%m/%d %H:%M')
    #timestamp += datetime.timedelta(seconds=time.timezone)
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

    path = (timestamp.strftime('%Y/%m') + '/' +
            os.path.splitext(os.path.basename(path))[0] + '.html')
    return Post(title=headers['subject'],
                timestamp=timestamp,
                path=path,
                content=html.decode('utf-8'),
                summary=headers.get('summary', ''))


def load_posts(root):
    posts = []
    def visit(arg, dirname, files):
        for file in files:
            _, ext = os.path.splitext(file)
            if ext == '.text':
                path = os.path.join(dirname, file)
                try:
                    post = load_post(path)
                except:
                    print 'when loading', path
                    raise
                posts.append(post)
    os.path.walk(root, visit, None)
    posts.sort(key=lambda post: post.timestamp)
    posts.reverse()
    return posts


def generate_archive(posts):
    def year_key(post):
        return post.timestamp.strftime('%Y')
    year_posts = []
    for year, posts in itertools.groupby(posts, key=year_key):
        year_posts.append({'year': year, 'posts': list(posts)})

    return jinja.get_template('archive.tmpl').render(
        root = './',
        title = settings['title'],
        grouped_posts = year_posts,
        pagetitle = '%s: archive' % settings['title'],
        )

def generate_post_page(root, post):
    return jinja.get_template('post.tmpl').render(
        root = root,
        title = settings['title'],
        extrahead = settings['index_extra_head'],
        post = post,
        url = os.path.join(root, post.path),
        pagetitle = '%s: %s' % (settings['title'], post.title),
        )


def generate_feed(posts):
    author = atom.Author(name=settings['author'], email=settings['email'])
    entries = []
    for post in posts[:5]:
        # Rewrite post path into the weird id form I used to use.
        id = settings['id_base'] + '/' + (
            post.timestamp.strftime('%Y-%m-%d') + '/' +
            os.path.splitext(os.path.basename(post.path))[0])
        timestamp = post.timestamp + datetime.timedelta(seconds=time.timezone)
        entries.append(atom.Entry(timestamp=timestamp,
                                  id=id,
                                  title=post.title,
                                  link=settings['link'] + post.path,
                                  content=post.content))
    feed = atom.Feed(title=settings['title'],
                     id=settings['id_base'],
                     link=settings['link'],
                     selflink=settings['link'] + 'atom.xml',
                     author=author,
                     entries=entries)
    return feed.to_xml()


def generate_frontpage(posts):
    posts = posts[:FRONTPAGE_POSTS]
    return jinja.get_template('frontpage.tmpl').render(
            title = settings['title'],
            extrahead = settings['index_extra_head'],
            posts = posts,
            )


def regenerate(in_dir):
    os.umask(022)

    global settings
    settings = util.parse_headers(util.read_file(
            os.path.join(in_dir, 'settings')).strip())

    global jinja
    jinja = jinja2.Environment(
        loader=jinja2.FileSystemLoader(os.path.join(in_dir, 'templates')),
        autoescape=True)

    posts = load_posts(os.path.join(in_dir, 'posts'))

    util.write_if_changed('index.html', generate_frontpage(posts))
    if len(posts) > FRONTPAGE_POSTS:
        util.write_if_changed('archive.html', generate_archive(posts))
    util.write_if_changed('atom.xml', generate_feed(posts))

    for post in posts:
        dir = os.path.split(post.path)[0]
        try:
            os.makedirs(dir)
        except OSError:
            pass
        root = '../' * post.path.count('/')
        util.write_if_changed(post.path,
                              generate_post_page(root, post).encode('utf-8'))


if __name__ == '__main__':
    regenerate(sys.argv[1])
