<p align="center">
	<img src='doc/docs/master/assets/images/Pachyderm_stacked--tagline--1200.png' height='225' title='Pachyderm'>
</p>

[![GitHub release](https://img.shields.io/github/release/pachyderm/pachyderm.svg?style=flat-square)](https://github.com/pachyderm/pachyderm/releases)
[![GitHub license](https://img.shields.io/badge/license-Pachyderm-blue)](https://github.com/pachyderm/pachyderm/blob/master/LICENSE)
[![GoDoc](https://godoc.org/github.com/pachyderm/pachyderm?status.svg)](https://pkg.go.dev/github.com/pachyderm/pachyderm/v2/src/client)
[![Go Report Card](https://goreportcard.com/badge/github.com/pachyderm/pachyderm)](https://goreportcard.com/report/github.com/pachyderm/pachyderm)
[![Slack Status](https://badge.slack.pachyderm.io/badge.svg)](https://slack.pachyderm.io)
[![CLA assistant](https://cla-assistant.io/readme/badge/pachyderm/pachyderm)](https://cla-assistant.io/pachyderm/pachyderm)

# Pachyderm – The Leader in Data Versioning and Pipelines for MLOps


Pachyderm is the leader in data versioning and pipelines for MLOps.  We provide the data foundation that allows data science teams to automate and scale their machine learning lifecycle while guaranteeing reproducibility. With investment from Benchmark,  Microsoft M12, and others, Pachyderm, Inc. offers a commercial Pachyderm Enterprise Edition and an open source Pachyderm Community Edition. Pachyderm helps customers get their ML and AI projects to market faster, lower data processing and storage costs, and supports strict data governance requirements.

## Features

- Automated Data Versioning — Pachyderm’s Data Versioning gives teams an automated and performant way to keep track of all data changes
- Data-Driven Pipelines — Pachyderm’s Containerized Pipelines speed data processing while lowering compute costs
- Immutable Data Lineage — Pachyderm’s Data Lineage provides an immutable record for all activities and assets in the ML lifecycle
- Console — The Pachyderm Console provides an intuitive visualization of your DAG (directed acyclic graph) and aids in reproducibility
- Notebooks — Pachyderm’s JupyterLab Mount Extension provides a point-and-click interface to Pachyderm versioned data
- Enterprise Administration — Pachyderm provides robust tools for deploying and administering Pachyderm at scale across different teams in your organization


## Getting Started
To start deploying your end-to-end version-controlled data pipelines, run Pachyderm [locally](https://docs.pachyderm.com/latest/getting-started/local-installation/) or you can also [deploy on AWS/GCE/Azure](https://docs.pachyderm.com/latest/deploy-manage/deploy/amazon_web_services/) in about 5 minutes. 

You can also refer to our complete [documentation](https://docs.pachyderm.com) to see tutorials, check out example projects, and learn about advanced features of Pachyderm.

If you'd like to see some examples and learn about core use cases for Pachyderm:
- [Examples](https://docs.pachyderm.com/latest/examples/examples/)
- [Use Cases](https://www.pachyderm.com/use-cases/)
- [Case Studies](https://www.pachyderm.com/case-studies/)

## Documentation

[Official Documentation](https://docs.pachyderm.com/)

## Community
Keep up to date and get Pachyderm support via:
- [![Twitter](https://img.shields.io/twitter/follow/pachyderminc?style=social)](https://twitter.com/pachyderminc) Follow us on Twitter.
- [![Slack Status](https://badge.slack.pachyderm.io/badge.svg)](https://slack.pachyderm.io) Join our community [Slack Channel](https://slack.pachyderm.io) to get help from the Pachyderm team and other users.

## Contributing
To get started, sign the [Contributor License Agreement](https://cla-assistant.io/pachyderm/pachyderm).

You should also check out our [contributing guide](https://docs.pachyderm.com/latest/contributing/setup/).

Send us PRs, we would love to see what you do! You can also check our GH issues for things labeled "help-wanted" as a good place to start. We're sometimes bad about keeping that label up-to-date, so if you don't see any, just let us know.

## Join Us

WE'RE HIRING! Love Docker, Go and distributed systems? Learn more about [our open positions](https://boards.greenhouse.io/pachyderm)

## Usage Metrics

Pachyderm automatically reports anonymized usage metrics. These metrics help us
understand how people are using Pachyderm and make it better.  They can be
disabled by setting the env variable `METRICS` to `false` in the pachd
container.

## License Information
Pachyderm has moved some components of Pachyderm Platform to a [source-available limited license](LICENSE). 

We remain committed to the culture of open source, developing our product transparently and collaboratively with our community, and giving our community and customers source code access and the ability to study and change the software to suit their needs.

Under the Pachyderm Community License, you can access the source code and modify or redistribute it; there is only one thing you cannot do, and that is use it to make a competing offering. 

Check out our [License FAQ Page](https://www.pachyderm.com/community-license-faq/) for more information.
