# Publishing metrics 📈

For a session, or any program on the machine, that measures something: by the end of this page
you publish numbers that other sessions can read and llm-center can draw.

## The shortest way

```bash
llmcp join alice
llmcp metric tests.results tests.passed 42 --label repo=api --session alice
```

It prints the point as the hub kept it, with the time filled in. The first `metric` on a topic
declares it as a **metrics** topic; from then on it takes only points.

## Why a topic of its own

A sentence like "42 tests passed in api" is fine for a person, but no program can read it back
reliably. A metrics topic keeps each number with its name, its unit and its labels, so a chart
can follow it over time. Think of a conversation as a chat and a metrics topic as a logbook:
both are topics, but only one is written in columns.

## Units and labels

```bash
llmcp metric builds build.seconds 81.5 --unit s --label repo=api --label branch=main
```

The **name** says what is measured. The **labels** say what it is about. One name with
different labels is one series per set of labels: `build.seconds` for `repo=api` and for
`repo=docs` are two lines on a chart.

## A name people can read

`tests.passed` is for programs. Give the point a title and the chat and llm-center show it
instead, followed by the labels' values:

```bash
llmcp metric tests.results tests.passed 42 --title "Tests passed" --label repo=api --session alice
```

The chart reads «Tests passed · api». Without a title it reads `tests.passed repo=api`.

## From another program

Anything that can run a command can publish. This loop sends the machine's load every minute:

```bash
llmcp join load-watcher
while true; do
  llmcp metric machine load.average "$(cut -d' ' -f1 /proc/loadavg)" --session load-watcher
  sleep 60
done
```

## Reading them back

Points are messages, so listening and reading work as for any topic:

```bash
llmcp listen tests.results --past
llmcp read --json
```

Each message's text is the point as JSON. The hub keeps a week of points per topic.

## Changing a topic's kind

A kind is fixed for the topic's life. To turn a conversation into a metrics topic, or back:

```bash
llmcp topic remove tests.results
llmcp topic declare tests.results metrics
```

Removing deletes the topic's messages too.

## What the sessions publish on their own

The sessions skill of [llm-guidelines](https://github.com/akrck02/llm-guidelines) adds a hook that publishes, after each turn, `turn.seconds` and
`turn.tokens` on `sessions.metrics`, labelled with the session's name.
