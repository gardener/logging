# PR Summarization Prompt

Summarize the changes in this pull request for a technical audience.
Highlight the main features improvements in one paragraph - required.
Add bug fixes if any in second paragraph - optional.
List breaking changes if any in third paragraph - optional.

Set the PR kind using the appropriate "/kind" identifiers. It shall be one of the following:
api-change|bug|cleanup|discussion|enhancement|epic|impediment|poc|post-mortem|question|regression|task|technical-debt|test
Set the PR area using the "/area logging" identifier.

Link any related issues using "Fixes #<issue_number>".

Provide a short release note, preserving the fence code block delimiters in the following format:

```<category> <target_group>
{{ release_note }}
```

Where:

- <category> is one of: breaking|feature|bugfix|doc|other
- <target_group> is one of: user|operator|developer|dependency

If no release note is required, just write "NONE" within the block.
