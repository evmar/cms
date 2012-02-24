#!/usr/bin/python

try:
    from xml.etree import ElementTree as ET
except ImportError:
    from elementtree import ElementTree as ET

import calendar
import datetime
import sys
import time
import markdown

def atomdate(dt):
    return dt.strftime('%Y-%m-%dT%H:%M:%SZ')
def entry_id(dt):
    unix_timestamp = calendar.timegm(dt.timetuple())
    return dt.strftime('tag:neugierig.org,%Y-%m-%d:' + str(unix_timestamp))
def htmldate(dt):
    return dt.strftime('%Y-%m-%d')

def write_atom(posts, outfile):
    feed = ET.Element('feed', xmlns='http://www.w3.org/2005/Atom')
    ET.SubElement(feed, 'title').text = 'neugierig.org updates'
    ET.SubElement(feed, 'id').text = 'http://neugierig.org/'
    ET.SubElement(feed, 'link', href='http://neugierig.org/')
    ET.SubElement(feed, 'link', rel='self',
                  href='http://neugierig.org/feed.xml')
    if posts:
        ET.SubElement(feed, 'updated').text = \
            atomdate(posts[0][0]['Timestamp'])
    author = ET.SubElement(feed, 'author')
    ET.SubElement(author, 'name').text = 'Evan Martin'
    ET.SubElement(author, 'email').text = 'martine@danga.com'
    for headers, body in posts:
        entry = ET.SubElement(feed, 'entry')
        timestamp = headers['Timestamp']
        # Adjust time to UTC by adding timezone offset.
        timestamp += datetime.timedelta(seconds=time.timezone)
        ET.SubElement(entry, 'id').text = entry_id(timestamp)
        ET.SubElement(entry, 'updated').text = atomdate(timestamp)
        ET.SubElement(entry, 'title').text = htmldate(timestamp)
        ET.SubElement(entry, 'content', type='html').text = body
    ET.ElementTree(feed).write(outfile)

def write_html(posts, outfile):
    print >>outfile, '<ul>'
    for headers, body in posts:
        timestamp = headers['Timestamp']
	assert body.startswith('<p>')
        body = body[:3] + '<b>%s</b> ' % htmldate(timestamp) + body[3:]
        print >>outfile, '<li>%s</li>' % body
    print >>outfile, '</ul>'

def load():
    posts = []
    f = open('sitefeed.txt', 'r')
    for post in f.read().split('====\n'):
        if not post:
            continue
        headertext, body = post.strip().split('\n\n', 1)
        headers = {}
        for header in headertext.split('\n'):
            key, val = header.split(': ', 1)
            if key == 'Timestamp':
                val = datetime.datetime.strptime(val, '%Y/%m/%d %H:%M')
            headers[key] = val
        body = markdown.markdown(body)
        posts.append((headers, body))
    f.close()
    return posts

if __name__ == '__main__':
    if len(sys.argv) < 2:
        print >>sys.stderr, 'usage: %s {html|feed}' % sys.argv[0]
        sys.exit(1)
    mode = sys.argv[1]
    assert mode in ('html', 'feed')
    posts = load()
    if mode == 'feed':
        write_atom(posts, sys.stdout)
    else:
        write_html(posts, sys.stdout)
