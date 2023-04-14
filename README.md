# Donation Page

This is a custom donation page for myself, but you can also use it if you want.

## Features

- ✅ Easy API to connect custom tools with this
- ✅ A permission system for applications
- ✅ PayPal checkout (including credit cards)
- ✅ Connecting Donor's discord account
- ✅ Multiple funds
- ✅ Fund Goals
- ✅ Fund aliases for quick links
- ✅ Monthly Pay Cycles
- ✅ Etc

## Config

This project is configured in 2 files - `.env` and `auths.json`

The `.env` file is for settings that will not be changed at runtime, and are options for setting the app up in general. To configure this, you can take a look at `template.env`, which has explanations for each configurable value. If there is a value already set (such as `PORT="3000"`), then the value that is set is the default value. This applies to empty values as well.

The `auths.json` file is for setting up external tools, allowing API connections. This file should contain a list of all applications. Note, each application **must** have a different token & name!

```json
[
  {
    "name": "Discord Bot",
    "permissions": 8,
    "token": "abc123"
  },
  {
    "name": "Twitch Relay",
    "permissions": 18,
    "token": "cba321"
  }
]
```

The permission value allows you limit the control of a certain application. For example, you don't want a twitch relay to create a new fund, do you? The permissions are as follow:

| Permission                              | Value |
| --------------------------------------- | :---: |
| Fetch the Discord Token                 |   2   |
| View life notifications                 |  16   |
| Admin fund control (create, edit, etc)  |  32   |
| Fetch historical donations              |  64   |
| Admin Permissions (ie. all permissions) |   8   |

To get the final permission value, simply add the values of all the permissions you want to add. For example, for the `Twitch Relay` app, I have it the permissions to view live notifications, and to fetch the discord token used by the donation API.

Finally, note the last permission, *8*. If an application has this, it will be able to execute any of the permissions above it. Consider it more of a shortcut.

## Setup

1. You must create a PayPal Live App. For that, go to [the developer console's application menu](developer.paypal.com/developer/applications)
2. Configure your paypal app, then go to webhooks & add `https://your-domain.com/api/PAYPAL_PATH`, where `PAYPAL_PATH` is pretty much any path you want. Consider this a password - do not share this path. For the webhook, enable `Checkout order approved`.
3. [Create a discord bot](https://discord.com/developers), allowing OAuth connections. Note it's token, client id and client secret.
4. Clone this repository into a folder 
5. Configure your `.env` and `auths.json` (see above). Remember for the `.env` - your `PAYPAL_PATH` is the path you used in step 2.
6. (Optional) add a favicon (see the )
7. Run `go build`
8. Run the resulting executable, named `donations` or `donations.exe`.

## Usage

Once the app is started, you can input commands into it! Currently only 2 are supported:

`reload` - This will reload the `auths.json` file. Use this if you want to add/remove application permissions, or applications in general

`fund` - Enter a CLI Input menu that will help with creating a new fund 

## Tools

Docs are a work in progress...

## Favicon

You can add a custom favicon by placing one in `pages/favicon.ico`

## Custom frontend

If you wanna do that, go for it. you can edit the `pages/*` files for that. You don't need to rebuild or restart the app between edits.

## API

Docs are a work in progress...