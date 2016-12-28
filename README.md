
GoNotify
========

####A simple program that uses `inotify` API in a Go environment.
(For now, **working only in Linux**.)

It looks recursively under a folder and warn you when a file or a folder has been created, deleted or moved.

It is in early stages...

> Usage: `gonotify FOLDER_PATH`

Next steps:

- Add to watch list new folder created.
- Libnotify implementation.