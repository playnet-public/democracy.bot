# democracy.bot
[![Go Report Card](https://goreportcard.com/badge/github.com/playnet-public/democracy.bot)](https://goreportcard.com/report/github.com/playnet-public/democracy.bot)
[![License: GPL v3](https://img.shields.io/badge/License-GPL%20v3-blue.svg)](https://www.gnu.org/licenses/gpl-3.0)
[![Build Status](https://travis-ci.org/playnet-public/democracy.bot.svg?branch=master)](https://travis-ci.org/playnet-public/democracy.bot)
[![Docker Repository on Quay](https://quay.io/repository/playnet/democracy.bot/status "Docker Repository on Quay")](https://quay.io/repository/playnet/democracy.bot)
[![Join Discord at https://discord.gg/dWZkR6R](https://img.shields.io/badge/style-join-green.svg?style=flat&label=Discord)](https://discord.gg/dWZkR6R)

A bot for making discord server democratic.

## Description

The whole idea for democracy.bot arose on a public developer discord.
We saw several servers, that are controlled by single admins enforcing their will upon the members. On the other hand, people don't want to leave because they want or need the community.

So solve this situation for us and to build something of value for communities, we started democracy.bot.
The bot should allow users to vote their admins for a certain time, control their decisions and act as a real community.

### Planed Features
* Suggesting admins to the Bot in DM before the vote starts
* Voting for admins by selecting the specific emote in the vote message
* Removing the old admin and adding the new one (or keeping the confirmed admin)
* Allow everybody to start a vote in the democracy channel for important topics
* Allow a distrust vote against the current admin
* Allow banned or kicked users to oppose the action to the democracy bot (DM) and let the community decide

### Assumptions
To realize those features, we have to take certain assumptions.
* The bot is the only one with full permission on the server
* The admin is not able to influence the democracy channel in any way
* The admin is not able to kick or ban users directly. Only via the bot
* The admin can not be kicked or banned. Only a distrust vote is possible
* The admin is not allowed to grant permissions directly. Only via the bot
* Votes have a time of expiration. Only the given votes count

## Dependencies
This project has a pretty complex Makefile and therefore requires `make`.

Go Version: 1.8

Install all further requirements by running `make deps`

## Usage

```
democracy.bot
```

## Development

This project is using a [basic template](github.com/playnet-public/gocmd-template) for developing PlayNet command-line tools. Refer to this template for further information and usage docs.
The Makefile is configurable to some extent by providing variables at the top.
Any further changes should be thought of carefully as they might brake CI/CD compatibility.

One project might contain multiple tools whose main packages reside under `cmd`. Other packages like libraries go into the `pkg` directory.
Single projects can be handled by calling `make toolname maketarget` like for example:
```
make template dev
```
All tools at once can be handled by calling `make full maketarget` like for example:
```
make full build
```
Build output is being sent to `./build/`.

If you only package one tool this might seam slightly redundant but this is meant to provide consistence over all projects.
To simplify this, you can simply call `make maketarget` when only one tool is located beneath `cmd`. If there are more than one, this won't do anything (including not return 1) so be careful.

## Contributions

Pull Requests and Issue Reports are welcome.
If you are interested in contributing, feel free to [get in touch](https://discord.gg/WbrXWJB)