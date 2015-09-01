#!/bin/bash

heroku pg:backups capture
curl -o backup/`date "+%Y-%m-%d"`.dump `heroku pg:backups public-url`
