+++
title =  "My Most Embarrassing Story as a Beginner Software Engineer"
date = 2023-07-09

[taxonomies]
tags = ["personal", "software"]

[extra]
photo = ""
+++

The beginner's curse of not knowing when to ask "why"

![Image](https://i.kym-cdn.com/entries/icons/facebook/000/008/342/ihave.jpg)

<!-- more -->

We all have stories of our huge gaps in understanding we had as junior developers, and mine is so embarrassing that I used to hesitate sharing it. The beginner's journey in software is often marked by confusion, impostor syndrome, and endless tutorials that leave us no better in understanding computers than before. I was privileged enough to have studied a formal degree in computer science, and while I felt unstoppable when I first started coding 10 years ago, my hubris left a lingering sting felt to this day. 

Back in 2013, as a freshman in college, I had built a small reputation on campus for having programmed a web-app called [Instanomz](https://www.thecrimson.com/flyby/article/2014/3/6/instanomz-brings-late-night-food/), a food delivery app for our campus. I became really good at following tutorials on how to build simple apps with Ruby on Rails, not even understanding how it actually worked. Yet, it was just enough knowledge to put something together that others could use, and so was Instanomz born. Armed with my go-to set of tutorials and free web UI templates, I felt I could build _anything_. 

A few months later, I somehow landed a side-gig helping out in building a full-stack web application. It was composed of a machine learning model for a recommendation engine written in Python, and a full-stack app with a shopping frontend written in Ruby.  Me, being decently familiar with building rails applications thanks to endless Ruby on Rails videos online, naively felt up to the task at the time.  I had just completed [CS50](https://pll.harvard.edu/course/cs50-introduction-computer-science), Harvard's popular course on computer science, and was deep into functional programming in OCaml thanks to CS51, its follow-up course. At the time, we had already learned a lot about theory, but we never learned how to build _real_ applications. 

The lead engineer had left the project and the team lead paid me to do the rest. My task was to get everything working by hooking up our ML model to the backend of our web app. **Not knowing how naive I was**, I began scouring the Internet for how a Ruby application could talk to Python code. Instead of writing a simple REST API between the two, and having the Python app as a separate server, I went off the deep end in reading about **foreign-function interfaces (FFI)**, CPython bindings, and even crazy concepts such as releasing the Ruby VM global lock during a Python function call context. All I wanted was for my backend to call some methods from Python to receive recommendations on user-submitted data, and had developed a one-track-mind into running both as the same process. In my mind, the fact that this was so _difficult_ felt ridiculous, and suddenly I even became convinced there was a need in the market for deep expertise in Ruby <> Python integrations given their popularity.

To add insult to injury, a few months before, [maintenance stopped on the FFI Ruby gem](https://news.ycombinator.com/item?id=6624468). I felt I was fighting a losing battle. I not only spent days working on understanding opaque type-passing, C native types, callback patterns to interact between the different bindings of the Python and Ruby applications, but also had never heard of those concepts before. When I reached my limit, I posted about my problem on various forums: **"How can I call Python code from Ruby?"**, and not a single person asked "why" I was trying to do this. In retrospect, they likely thought I was an expert exploring deep integrations between programming languages. Then again, my question did not provide any context of my problem, as one of my biggest flaws at the time was how easy I developed tunnel vision into the "how" and not the "why".

```ruby
module A
  extend FFI::Library
  ffi_lib 'c'
  attach_function :strlen, [:string], :int
end
A.strlen("abc")
# 3
```

My project's boss, not knowing how big of a mess I had gotten myself into, soon worried about how long it was taking to integrate the two applications. He thought he could help me out, so he reached out to his sibling that was a professional software engineer at the time. His response was: 

> "Is it that hard to make a REST API to your Python server? Monoliths are popular these days thanks to Ruby on Rails, yes, but I'd recommend running your Python server separately and talking to it via HTTP"

...(_crickets_)

My brain turned to mush, and there was nothing else I wanted to do than to dig a hole in the ground and bury my head in it. In that span of time, I:

- Read almost an entire book on foreign function interfaces
- Explored endless forums on adding foreign library bindings to Ruby code
- Went down the rabbit hole of CPython's internals
- Thought I was going crazy and that this was an untapped, market opportunity
- Thought about starting a _company_ that helps Ruby on Rails apps interact with other language apps

...when all I had to do was send a JSON POST request to my Python app, have it do its thingamajig, and respond with its recommendation data. At the time, however, I neither took a step back and framed my question in context to others, *nor* did I evaluate alternatives to my problem before going deep into it. Was the fact that computer science programs do not teach practical knowledge part of the problem? I'd say yes, but it's not entirely their fault compared to my own. If there's anything to takeaway from this story, it's that **knowing when to ask "why"** was my biggest asset in growing from a naive beginner into somebody that understands computers a little bit better today.