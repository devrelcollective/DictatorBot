# DictatorBot

DictatorBot is used by the [DevRelCollective](https://devrelcollective.fun) to manage which moderator (Benevolent Dictator) is on-call at any given time.

It is built in GOLANG, using Camunda Platform BPM as an orchestration platform.

It was written by [David G. Simmons](https://github.com/davidgs).

## Configuration and building

Before attempting to run the Dictator bot you should make sure that you have created a Slack Application, that it has the proper permissions to write to channels and Direct Messages, etc.

App your APP_ID, TOKEN, Verification Secret, and Channel ID to the `dictator.yaml` file.

You will need to edit `dictator.yaml` for your installation and make sure that the `dictators`, etc. are all correct.

```
% go get
% go build dictator.go
% ./dictator
```
From that point on, you can use DictatorBot in your Slack Group.

## Deploying Dictator Bot

You will need an instance of [Camunda Platform BPM](https://camunda.com) running somewhere. It *must* be running on a secure (TLS/HTTPS) server, as *must* the `dictator` process described above.

Deploy the `DictatorBot.bpmn` file to your Camunda Platform instance.

Make sure that you have edited the proper server names in the `dictator.yaml` file to point to your servers.

## Using Dictator Bot

Available commands are:
 1) `help` or `?`
 2) `rotate` or `rotation` to get the full rotation schedule
 3) `who` to see who the current on-call person is
 4) `next` to see who the next on-call person will be
 5) `@username` to place someone on-call
 6) `auth` or `authorized` to see who is authorized to use the DictatorBot
 7) `update` to place the next person in the rotation on-call

The DictatorBot can _only_ be called in channels where it has been invited, or in Direct Messages, and can _only_ be called by those listed in the `dictator.yaml` file. 