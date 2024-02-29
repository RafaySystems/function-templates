# function-templates
Templates for Environment Manager Functions


# Development Readme
## Nix
Nix is used to setup reproducible development environments.
* Follow instructions at [Install](https://nixos.org/download.html#nix-install-macos) to install nix.
* Nix shell [configuration](./shell.nix) should be updated when new tools are added.

## Direnv
Direnv allows automatic loading/unloading of development environment.
* Follow instructions at [Install](https://direnv.net/docs/installation.html) to install direnv.
* Create `.envrc` with the following content 
```
use_nix
```
* Run `direnv allow .` to allow loadin of `.envrc` in the current folder.
* `.envrc` is added to `.gitignore`, so it wont be checked in.
* Secrets required for running unit/integration tests can be added to `.envrc`

