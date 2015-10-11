# MontyPythonAndTheHolyGrailBot
A Monty Python Telegram Bot Written in Go.

The bot is still running on Telegram, @MontyPythonAndTheHolyGrailBot

Discontinued Project.

This was something I wrote over the course of a few hours on the weekend.  It uses BoltDB for the database and saves 8ball.txt and lyrics.txt to different buckets.  The bot will respond randomly with a line of text from quotes.txt and then give you your fortune at the bottom of the message (from 8ball.txt).

It would be possible to compile this yourself and run it anywhere, given that you have generated an API key from telegram and set up the environment variable on your computer "MONTYPYTHONBOT".  This variable needs to have a portion of the telegram URL to make polling requests.

Open source.  Free to use, distribute, and modify.
