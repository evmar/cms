#!/usr/bin/python

try:
    from xml.etree import ElementTree as ET
except ImportError:
    from elementtree import ElementTree as ET

def atomdate(dt):
    return dt.strftime('%Y-%m-%dT%H:%M:%SZ')

class Entry(object):
    def __init__(self, timestamp, id, title, link, content):
        self.timestamp = timestamp
        self.id = id
        self.title = title
        self.link = link
        self.content = content

    def to_et(self):
        entry = ET.Element('entry')
        ET.SubElement(entry, 'id').text = self.id
        ET.SubElement(entry, 'updated').text = atomdate(self.timestamp)
        ET.SubElement(entry, 'title').text = self.title
        if self.link:
            ET.SubElement(entry, 'link', href=self.link)
        ET.SubElement(entry, 'content', type='html').text = self.content
        return entry


class Author(object):
    def __init__(self, name, email):
        self.name = name
        self.email = email

    def to_et(self):
        author = ET.Element('author')
        ET.SubElement(author, 'name').text = self.name
        ET.SubElement(author, 'email').text = self.email
        return author

class Feed(object):
    def __init__(self, title, id, link, selflink, author, entries):
        self.title = title
        self.id = id
        self.link = link
        self.selflink = selflink
        self.author = author
        self.entries = entries

    def to_et(self):
        feed = ET.Element('feed', xmlns='http://www.w3.org/2005/Atom')
        ET.SubElement(feed, 'title').text = self.title
        ET.SubElement(feed, 'id').text = self.id
        ET.SubElement(feed, 'link', href=self.link)
        ET.SubElement(feed, 'link', rel='self', href=self.selflink)
        ET.SubElement(feed, 'updated').text = (
            atomdate(self.entries[0].timestamp))
        feed.append(self.author.to_et())
        for entry in self.entries:
            feed.append(entry.to_et())
        return feed

    def to_xml(self):
        # 'unicode' is not an encoding, grumble.
        return ET.tostring(self.to_et(), encoding='unicode')
