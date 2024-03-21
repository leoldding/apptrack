# Apptrack (Application Tracker)

`apptrack` is a CLI tool for tracking job applications.
It automatically parses LinkedIn jobs and adds it to a Notion database.
Can also manually fill in information for other websites.

## Installation 

Use go to install from source.
```
go install github.com/leoldding/apptrack
```

## Setup

### Database

On a new Notion page, create a database with the following property names and types:
* Company (title)
* Position (text)
* Location (text)
* Date Applied (date)
* Status (status)
    * To-do
        * Ready to apply (default)
    * In progress
        * Applied
* Link (text)

Feel free to add other properties for your own database such as `Accepted` to the `Complete` sub-property under `Status`. 

### Database ID

Make sure that you have the database opened on as a full page. Use the `Share` menu on top to `Copy link`. 
Paste the link somewhere so that you can inspect it. 

The link should be in the following format:
```
https://www.notion.so/{workspace_name}/{database_id}?v={view_id}
```

Keep track of the part in the link that corresponds to `{database_id}`.

### Integration

Follow the first three sections under `Getting Started` in this [Notion guide](https://developers.notion.com/docs/create-a-notion-integration#getting-started) to create a Notion integration, get your API key, and giving the integration permissions to your database.

### Variables

Now we will use the database ID and API key from the two previous steps.

Add `APPTRACK_NOTION_DATABASE_ID=<your-database-id>` and `APPTRACK_NOTION_API_KEY=<your-api-key>` to your `~/.bashrc` or `~./zshrc` file in order for the tool to work.

## Usage

### LinkedIn

For LinkedIn jobs, simply call the tool using `apptrack` and paste the job link when prompted.
If there is any information that was not found, you will be prompted to fill it in.

### Other Jobs

For other jobs, use the `-manual` or `-m` flag.
You will simply be prompted for the necessary information to fill in the database row.

### Saving Jobs

If you haven't applied to a job, you can use the `-save` or `-s` flag. 
It will set the `Status` property to the default value of `Ready to apply` and not `Applied`.
