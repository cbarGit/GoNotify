
GoNotify
========

####A simple program that uses `inotify` API in a Go environment.
(For now, **working only in Linux**.)

It looks recursively under a folder and warn you when a file or a folder has been created, deleted or moved.

Library `libnotify` of your distribution needed.

For notification through `libnotify` has been used the library from:
https://github.com/mqu/go-notify

> Usage: `gonotify FOLDER_PATH`

Next steps:

- Add to watch list new folder created. --> done.
- Libnotify implementation. --> done.
- Handle "rename" file/folder problem. Now alert only with `IN_MOVED_FROM` and `IN_MOVED_TO`.
- GUI Implementation?
