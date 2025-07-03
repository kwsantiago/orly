# lol

location of log

This is a very simple, but practical library for logging in applications. Its
main feature is printing source code locations to make debugging easier.

## terminals

Due to how so few terminals actually support source location hyperlinks, pretty much tilix and intellij terminal are
the only two that really provide adequate functionality; this logging library defaults to output format that works
best with intellij. As such, the terminal is aware of the CWD and the code locations printed are relative, as
required to get the hyperlinkization from this terminal. 

Handling support for Tilix requires more complications and
due to advances with IntelliJ's handling it is not practical to support any other for this purpose. Users of this
library can always fall back to manually interpreting and accessing the relative file path to find the source of a log.

## using with tilix

this enables us to remove the base of the path for a more compact code location string,
this can be used with tilix custom hyperlinks feature

create a script called `setcurrent` in your PATH ( eg ~/.local/bin/setcurrent )

    #!/usr/bin/bash
    echo $(pwd) > ~/.current

make it executable

    chmod +x ~/.local/bin/setcurrent

set the following environment variable in your ~/.bashrc

    export PROMPT_COMMAND='setcurrent'

using the following regular expressions, replacing the path as necessary, and setting
perhaps a different program than ide (this is for goland, i use an alias to the binary)

      ^((([a-zA-Z@0-9-_.]+/)+([a-zA-Z@0-9-_.]+)):([0-9]+))    ide --line $5 $(cat /home/mleku/.current)/$2
      [ ]((([a-zA-Z@0-9-_./]+)+([a-zA-Z@0-9-_.]+)):([0-9]+))  ide --line $5 $(cat /home/mleku/.current)/$2
      ([/](([a-zA-Z@0-9-_.]+/)+([a-zA-Z@0-9-_.]+)):([0-9]+))  ide --line $5 /$2

and so long as you use this with an app containing /lol/log.go as this one is, this finds
that path and trims it off from the log line locations and in tilix you can click on the
file locations that are relative to the CWD where you are running the relay from. if this
is a remote machine, just go to the location where your source code is to make it work