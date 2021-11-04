# Laihfe
Something to inspire myself to get more organized and learn go.
For now, todo manager. In the future, useful!

![](./screenshot.png)

## Usage
Currently only clone and run. In the future, a package!
```
git clone https://github.com/sinakhalili/laihfe
cd laihfe
go run main.go
# to install it to ~/go/bin/ or GOPATH
go build
go install
```

## How it works
Todos are read and stored in a plain text json file `.todos.json`. 

Type `i` to go into insert mode and write a todo. 

`esc` will put you into normal mode where you can toggle todos on and off with `enter`. 
You can move the todo cursor with `j` and `k`. 

To delete a todo, move the cursor in normal mode and press `d`.

You can undo a deletion with `u`.

To quit, press `q` in normal mode.

## Keep it top of mind
Add the `laihfe -l` command at the end of your `.bashrc`/`.zshrc` to
spit out your current todos. That way you can see them every time
you open a terminal!

Uses the wonderful [bubbletea](https://github.com/charmbracelet/bubbletea) package for the TUI. 
