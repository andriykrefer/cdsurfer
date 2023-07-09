
# 💿🌊 cd-surfer
`cd-surfer` is a replacement for the endless `cd` `ls` `cd` `ls` `...` cycle on linux shells. It is meant to be a simple and fast TUI file manager.

Some features:
- No dependencies, single binary
- Fast navigation, less keytrokes than cd / ls
- Easy in-folder search (just start typing)
- Should be easy to integrate with any shell (currently it only supports bash)

![demo](https://github.com/andriykrefer/cdsurfer/assets/30701181/39df95b6-d80a-4de3-8b16-12684e9f574d)

## Install
- Run
```bash
wget -q -O - https://raw.githubusercontent.com/andriykrefer/cdsurfer/master/install.sh | bash
```
- Restart the terminal to apply the config

#### Alternative Manual Install
- Download a pre-build binary in the releases section or compile it yourself with with `cd cmd/cd-surfer/ && go build .`
- Move to /bin/cd-surfer with 755 permission
- Add the following lines to your `~/.bashrc`:
```bash
function s {
  eval "$(/bin/cd-surfer "$@")"
}
```
- Restart the terminal to apply

## Usage
Run the command `s`

### Keybinds
- `Arrow keys`, `PageUp`, `PageDown`, `Home` and `End` to navigate
- `Enter` and `Tab` to enter directory
- `Alt+Backspace` to go to the parent folder
- `/` to manually input the desired path
- Just start typing (`a-z`, lowercase) to search inside folder. The search is case-insensitive.
- `Alt+q` to quit WITHOUT changing directory on the parent shell
- `Ctrt+c` or Esc to quit CHANGING directory on the parent shell
- `~` to go to home directory
- `-` to go to home directory
- `Ctrl+u` to clear the input (same as bash)

## How it works
Unfortunately, there is easy way to directly change directories of the parent shell of a spawned process. The way `cd-surfer` works is outputting the `cd` command as a string, so the parent shell can evaluate it and change the directory.

## TODO features/bugs
Despite that it is still lacking a lot o features I want in the, for a weekend projectec it is good enought to use in my daily routine. Besides, I do not want to make it a bloated software, so I will keep it simple, focusing on its core features.

Below is a list of its "roadmap":

- ~~Basic functions (cd and ls replacement)~~ ✔
- ~~Search~~ ✔
- ~~Manual path input~~ ✔
- ~~Finish detailed file view~~ ✔
- Better README.md
- Add help/info and tips in the software
- File operations
    - Copy
    - Move
    - Delete
    - Rename
    - Create folder

- Open Editor when select a file
- Create a config file for customizations
- Handle Symlinks
- Easy permissions editor
- Handle errors gracefully. For instance when a user tries to access a folder he does not have permissions
- Support for other shells besides bash
- Support for Windows and Mac
- Tabs
