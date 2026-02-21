# Preface

## My Story

In 2005, I wrote my first line of PHP. I was hooked immediately. Over the next seventeen years, PHP became more than a programming language—it became the lens through which I saw software development. I lived through PHP 4's procedural chaos, PHP 5's object-oriented renaissance, and PHP 7's performance revolution. I built applications with Symfony from version 1.0 onwards, watching it mature into one of the most elegant frameworks in any language.

By 2022, PHP and I had developed a kind of telepathy. I could feel when code was right. I knew, without thinking, exactly how to structure a service, wire a dependency, or craft a clean controller. The language had become an extension of my thoughts.

Then I started writing Go.

## The Uncomfortable Truth

Learning Go's syntax took a few weeks. Learning to think in Go has taken years—and I'm still not there.

This isn't about intelligence or experience. It's about rewiring seventeen years of deeply ingrained mental models. Every time I reach for inheritance, Go reminds me it doesn't exist. Every time I want to throw an exception, I must write `if err != nil`. Every time I expect magic, I find explicit wiring.

The transition has been humbling. And illuminating.

## Why This Book Exists

Most Go books teach you Go. This book teaches you how to stop thinking in PHP.

If you've spent years mastering PHP—especially in the Symfony ecosystem—you've developed powerful mental models. These models served you well. But they're now fighting against Go's philosophy at every turn.

This book is not a beginner's guide. It assumes you can already write Go code that compiles and runs. What you might not be able to do is write *idiomatic* Go—code that feels natural to Go developers, code that leverages Go's strengths instead of fighting them.

We'll examine every mental model you've built in PHP and show you its Go equivalent (or lack thereof). We'll explore why certain patterns feel wrong in Go, and how to develop new instincts that feel right.

## Who This Book Is For

You should read this book if:

- **You've mastered PHP**, especially with frameworks like Symfony
- **You've started learning Go**, but it doesn't feel natural yet
- **You keep reaching for PHP patterns** that don't exist in Go
- **You want to understand Go's philosophy**, not just its syntax
- **You're frustrated** that years of experience seem to slow you down

You should probably look elsewhere if:

- You're new to programming entirely
- You've never worked with PHP seriously
- You're already comfortable writing idiomatic Go

## What You'll Learn

**Part I: The Mental Shift** examines the philosophical differences between PHP and Go. We'll explore why your PHP brain fights Go and how to make peace with the transition.

**Part II: Structural Rewiring** covers the fundamental building blocks—structs instead of classes, composition instead of inheritance, interfaces that work implicitly.

**Part III: Practical Patterns** takes you through real-world concerns: web development, databases, APIs, testing, and configuration—all from a PHP developer's perspective.

**Part IV: Concurrency** introduces Go's killer feature—something PHP simply doesn't have. We'll build new mental models from scratch.

**Part V: Advanced Topics** covers reflection, performance optimisation, and system programming.

**Part VI: Deployment and Migration** provides practical strategies for building, deploying, and migrating from PHP to Go.

## How to Read This Book

Each chapter compares PHP and Go approaches side by side. We'll show Symfony patterns you know intimately, then demonstrate their Go equivalents (or explain why no equivalent exists).

Code examples assume familiarity with modern PHP (8.x) and Symfony (5.x/6.x). Go examples target Go 1.21+.

The exercises at the end of each chapter aren't optional. They're designed to break your PHP habits and build Go instincts. Do them.

## A Note on Difficulty

This transition is hard. Not because Go is complex—it's famously simple. But because you're not learning something new; you're unlearning something old while learning its replacement.

Be patient with yourself. The discomfort you feel is the learning happening.

## Acknowledgements

To the PHP community that shaped my thinking for seventeen years. To the Go community that's reshaping it now. And to everyone who's ever felt like an expert beginner—starting over in a new language, humbled by how much they have to relearn.

Let's begin.

---

*"In the beginner's mind there are many possibilities, but in the expert's there are few."* — Shunryu Suzuki
