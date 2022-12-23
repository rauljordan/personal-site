+++
title =  "How My Team Won the Facebook GraphQL Hackathon"
date = 2017-02-04

[taxonomies]
tags = ["web-development"]
+++

```js
links: {
  type: new GraphQLList(HtmlPage),
  resolve: async (root, args, context) => {
    const res = await fetch(root.url);
    const $ = cheerio.load(await res.text());
    const links = $('a').map(function() {
      if (!$(this).attr('href')) {
        return;
      }
      if ($(this).attr('href') !== '#' && $(this).attr('href').indexOf('http') > -1) {
        return $(this).attr('href');
      }
    }).get();

    return links.map(url => ({ url }))
  }
},
```

A few months ago, my teammate Trey Granderson who worked with me at Kynplex, suggested we go to Facebook’s official GraphQL hackathon at their Cambridge headquarters, where we put together a simple project to try to use the awesomeness of the language to create something nifty and meet new people.

<!-- more -->

GraphQL is a specification created by Facebook for a query language that makes it easy to load data into any application. It was developed by the Facebook team to easily write API calls in a way that makes sense with how social networks are structured.

For example, a user on Facebook can have many friends, and those friends each have more friends, likes, followers, photos, and more. Having to do tons of nested API calls for each of these entities and also having to parse those responses is an incredibly annoying pain point for many similar webapps. GraphQL completely changed that landscape!

With GraphQL, developers are able to write a simple query string for an endpoint that reads kinda like simplified JSON according to a functions called resolvers that interact with a database and contain the business logic that actually fetches the data.

For example, if you have a GraphQL endpoint that returns the currently signed in user, you can fetch it as follows:

```js
const query = gql`
  query MyExampleQuery {
    currentUser {
      name
      photo
    }
  }
`;
```

Then, we can plug it into a react component and render that data however we want.

```js
import { graphql, compose } from 'react-apollo';

function UserProfile(props) {
  return 'Your Name is: {props.data.currentUser.name}';
}

export default compose(graphql(query))(UserProfile);
```

A cool thing about GraphQL is that it allows you to define how all of your models interact with each other and how they are connected. It basically gives you the ability to create an API Graph for your application. For example, the return type of the currentUser query is the User type, and the return type of the friends query is also a User type, allowing us to do fancy things like:

```js
const query = gql`
  query MyExampleQuery {
    currentUser {
      name
      photo
      friends(limit: 10) {
        name
        photo
        friends(limit: 10) {
          name
          photo
        }
      }
    }
  }
`;
```

The recursive properties of GraphQL make it incredibly powerful not just as a language to query stuff from a database, but also as an interesting tool that entire projects can be built upon.

## We Decided to Build a Web Scraper

After a lot of thought, we wanted to create a tool that would be useful to anyone and would leverage the recursive power of GraphQL to solve a problem in an interesting way. So we figured, why not build a web scraper that uses GraphQL? You could have a single query resolver called scrape that takes in a URL as a parameter and returns a generic defined entity such as an HtmlNode.

We wanted to make this tool fun to use and allow anyone to query for complex pieces of the DOM of any website by using graphql queries that are generated on the fly. Ideally, we wanted to be able to do something like:

```js
{
  scrape(url: 'https://google.com') {
    div(id: 'header') {
      ul {
        li {
          a {
            href
            content
          }
        }
      }
    }
  }
}
```

This gives us a very visual representation of what’s being scraped from the page. However, we wanted to go even one step further with recursiveness and see if we could directly scrape all of the hyperlinks on a given page and fetch their info as well.

```js
{
  scrape(url: 'https://google.com') {
    links {
      title
      footer {
        content
      }
    }
  }
}
```

Giving us the ability to directly scrape the DOM from any external links in google’s home page! Extend this even more and we’d be able to scrape the entire Internet (well, not really but you get the idea :P)

## So How Did We Do It?

We needed to leverage recursive types and some metaprogramming in the GraphQL.js npm package to get this working (link to the full repo is at the end).

First, we setup a simple express server that has a graphql endpoint over HTTP that we can access directly through REST or through GraphiQL (a handy IDE explorer that Facebook created for us).

```js
import express from 'express';
import cors from 'cors';
import graphqlHTTP from 'express-graphql';

import schema from './schema';

const app = express();
app.use(cors());

app.use('/', (req, res) => {
 graphqlHTTP({
   schema: schema,
   pretty: true,
   graphiql: true,
 })(req, res);
});

const port = 3010;

app.listen(port, () => {
 console.log(`app started on port ${port}`);
});
```

In the code above, we imported our schema that defines all the types and queries in our GraphQL API and attached it to a library called graphqlHTTP. Let’s take a look at this schema file.

```js
import {
  graphql,
  GraphQLSchema,
  GraphQLObjectType,
  GraphQLString
} from 'graphql';

import { HtmlPage } from './resolvers';

var schema = new GraphQLSchema({
  query: new GraphQLObjectType({
    name: 'Query',
    fields: {
      scrape: {
        type: HtmlPage,
        // `args` describes the arguments that the `scrape` query accepts
        args: {
          url: { type: GraphQLString }
        },
        resolve: function (_, { url }) {
          return {
            url
          };
        }
      }
    }
  }),
});

export default schema;
```

Here we define the main Query type, which contains a single resolver called scrape. This is a query that returns an entity of type HtmlPage, takes in a url string as an argument, and allows us to continue writing subqueries from there. This is basically all we needed to get this started, but the really awesome part is in the actual HtmlPage recursive type.

Let’s see what fields and queries that HtmlPage has available to it by looking at our final file: the resolvers of our GraphQL API. We’ll take it one step at a time, starting with some basic imports.

```js
import cheerio from 'cheerio';
import fetch from 'node-fetch';

import {
  graphql,
  GraphQLSchema,
  GraphQLObjectType,
  GraphQLString,
  GraphQLInt,
  GraphQLList
} from 'graphql';
```

We need to define the actual base return type of our scrape query, which is the HtmlPage type. Initially, we want it to have some basic fields such as the url of the page, the hostname, and the title.

```js
export const HtmlPage = new GraphQLObjectType({
  name: 'HtmlPage',
  fields: () => ({
    url: {
      type: GraphQLString,
      resolve(root, args, context) {
        return root.url;
      }
    },
    hostname: {
      type: GraphQLString,
      resolve(root, args, context) {
        const match = root.url.match(/^(https?\:)\/\/(([^:\/?#]*)(?:\:([0-9]+))?)([\/]{0,1}[^?#]*)(\?[^#]*|)(#.*|)$/);
        return match && match[3];
      }
    },
    title: {
      type: GraphQLString,
      resolve: async (root, args, context) => {
        const res = await fetch(root.url);
        const $ = cheerio.load(await res.text());
        return $('title').text();
      }
    }
  })
});
```

It would be cool if we could also fetch all the images on the page, so let’s add a field for that as well:

```js
export const HtmlPage = new GraphQLObjectType({
  name: 'HtmlPage',
  fields: () => ({
    images: {
      type: new GraphQLList(GraphQLString),
      resolve(root, args, context) {
        return getImgsForUrl(root.url);
      }
    },
    // all other fields below...
    ...
});

const getImgsForUrl = async (url) => {
  const res = await fetch(url);
  const $ = cheerio.load(await res.text());
  return $('img').map(function() { return $(this).attr('src'); }).get();
};
```

Now, we want to create a resolver for every valid DOM element that we can call on this HTMLPage type. For example, we want to be able to write:

```js
{
  scrape('https://facebook.com') {
    div {
      span {
        content
      }
    }
  }
}
```

This will allow us keep nesting our queries as much as we need to extract any content from a page. So let’s define our HtmlNode type first.

```js
const validHtmlTags = [
  'div'
];

const validAttributes = {
  id: { type: GraphQLString },
  class: { type: GraphQLString },
  src: { type: GraphQLString },
  content: { type: GraphQLString },
};

const HtmlNode = new GraphQLObjectType({
  name: 'HtmlNode',
  fields: htmlFields,
});
```

Now we need to define every field that the HtmlNode type will have, so we’ll do it by extending the validHtmlTags array to include any other tags we want and then converting that into an array of GraphQL resolvers as follows:

```js
const htmlFields = () => validHtmlTags.reduce((prev, tag) => ({
  ...prev,
  [`${tag}`]: {
    type: HtmlNode,
    args: {
      ...validAttributes,
    },
    resolve(root, args, context) {
      const tagHistory = {
        tag,
        args,
        url: root.url || root[0].url,
      }
      return [...root, here];
    }
  },
  // some more stuff...
  ...
}), {});
```

Let’s break down what’s going on here. We add a resolver for each tag in the validHtmlTags array that has a type of HtmlNode and a valid set of arguments (id, class, etc.) so we can do fancier and more specific DOM scraping. In order to finally fetch the content inside of this node, we need to keep track of all the other nodes above it so we can use tools such as Cheerio to recursively fetch their content.

We save the type of tag (div, span, etc.) and the arguments to its resolvers inside of an array called tagHistory that we then return in our resolver.

This means that when we query for

```js
{
  scrape(url: 'google.com')
    div {
      div {
        span {
          content
        }
      }
    }
  }
}
```

We’ll be storing this nested query as an array that would look like:

```js
const tagHistory = [ { tag: 'div' }, { tag: 'div' }, { tag: 'span' }]
```

Making it easy for us to parse this in our base case, a resolver called content that would return the text content inside of a certain HtmlNode. Let’s see how that would work.

```js
const htmlFields = () => validHtmlTags.reduce((prev, tag) => ({
  ...prev,
  [`${tag}`]: {
    type: HtmlNode,
    args: {
      ...validAttributes,
    },
    resolve(root, args, context) {
      const tagHistory = {
        tag,
        args,
        url: root.url || root[0].url,
      }
      return [...root, here];
    }
  },
  content: {
    type: GraphQLString,
    resolve: async (root, args, context) => {
      const res = await fetch(root[0].url);
      const $ = cheerio.load(await res.text());
      const selector = root.reduce((prev, curr) => {
        let arg = '';
        if(curr.args.id) {
          arg = `#${curr.args.id}`;
        } else if (curr.args.class) {
          arg = `.${curr.args.class}`;
        }

        const ret = `${prev} ${curr.tag}${arg}`;
        return ret;
      }, '');
      return $(selector).html();
    }
  }
}), {});
```

The content resolver is the base case of our recursion. Here, root is going to be the tagHistory array that we created through all the nested queries, so we simply use node-fetch to make a GET request to the root URL we want to scrape.

Then, we load it into cheerio, which is a JQuery library that works on the server. This reduce basically reduce the entire tagHistory array into a single JQuery selector string.

In this case, reducing the tagHistory array would give us the following string

`'div div span'`

and if we passed in the ID parameter or class parameter, we would get something like

`'div.navbar-container div.navbar-wrapper span.brand-logo'`

At the very end, we simply call the JQuery function html() to fetch the inner content of the selector we created! It doesn’t stop there though…we finally need to plug this into the HtmlPage type we created in the beginning so that we’re able to query for these DOM elements. Here is the new HtmlPage type with its resolvers!

```js
export const HtmlPage = new GraphQLObjectType({
  name: 'HtmlPage',
  fields: () => ({
    // We add the htmlFields resolvers and types that we defined above
    ...htmlFields(),
    images: {
      type: new GraphQLList(GraphQLString),
      resolve(root, args, context) {
        return getImgForUrl(root.url);
      }
    },
    url: {
      type: GraphQLString,
      resolve(root, args, context) {
        return root.url;
      }
    },
    hostname: {
      type: GraphQLString,
      resolve(root, args, context) {
        const match = root.url.match(/^(https?\:)\/\/(([^:\/?#]*)(?:\:([0-9]+))?)([\/]{0,1}[^?#]*)(\?[^#]*|)(#.*|)$/);
        return match && match[3];
      }
    },
    title: {
      type: GraphQLString,
      resolve: async (root, args, context) => {
        const res = await fetch(root.url);
        const $ = cheerio.load(await res.text());
        return $('title').text();
      }
    }
  })
});
```

To put a cherry on top, we added a resolver that would allow us to scrape all the hyperlinks in a page and return HtmlPage entities that we would also be able to run fancy queries on! Here’s how we did it:

```js
links: {
  type: new GraphQLList(HtmlPage),
  resolve: async (root, args, context) => {
    const res = await fetch(root.url);
    const $ = cheerio.load(await res.text());
    const links = $('a').map(function() {
      if (!$(this).attr('href')) {
        return;
      }
      if ($(this).attr('href') !== '#' && $(this).attr('href').indexOf('http') > -1) {
        return $(this).attr('href');
      }
    }).get();

    return links.map(url => ({ url }))
  }
},
```

Putting it all together, here’s our final resolver file:

```js
import cheerio from 'cheerio';
import fetch from 'node-fetch';

import {
  graphql,
  GraphQLSchema,
  GraphQLObjectType,
  GraphQLString,
  GraphQLInt,
  GraphQLList
} from 'graphql';

const validAttributes = {
  id: { type: GraphQLString },
  class: { type: GraphQLString },
  src: { type: GraphQLString },
  content: { type: GraphQLString },
};

export const validHTMLTags = [
  'div',
  'span',
  'img',
  'body',
  'a',
  'b',
  'i',
  'p',
  'h1',
  'h2',
  'h3',
  'article',
  'footer',
  'form',
  'input',
  'ul',
  'li'
];

const htmlFields = () => validHTMLTags.reduce((prev, tag) => ({
  ...prev,
  [`${tag}`]: {
    type: HtmlNode,
    args: {
      ...validAttributes,
    },
    resolve(root, args, context) {
      const here = {
        tag,
        args,
        url: root.url || root[0].url,
      }
      return [...root, here];
    }
  },
  content: {
    type: GraphQLString,
    resolve: async (root, args, context) => {
      const res = await fetch(root[0].url);
      const $ = cheerio.load(await res.text());
      const selector = root.reduce((prev, curr) => {
        let arg = '';
        if(curr.args.id) {
          arg = `#${curr.args.id}`;
        } else if (curr.args.class) {
          arg = `.${curr.args.class}`;
        }

        const ret = `${prev} ${curr.tag}${arg}`;
        return ret;
      }, '');
      return $(selector).html();
    }
  }
}), {});

const HtmlNode = new GraphQLObjectType({
  name: 'HtmlNode',
  fields: htmlFields,
});

const getImgsForUrl = async (url) => {
  const res = await fetch(url);
  const $ = cheerio.load(await res.text());
  return $('img').map(function() { return $(this).attr('src'); }).get();
};

export const HtmlPage = new GraphQLObjectType({
  name: 'HtmlPage',
  fields: () => ({
    ...htmlFields(),
    images: {
      type: new GraphQLList(GraphQLString),
      resolve(root, args, context) {
        return getImgsForUrl(root.url);
      }
    },
    links: {
      type: new GraphQLList(HtmlPage),
      resolve: async (root, args, context) => {
        const res = await fetch(root.url);
        const $ = cheerio.load(await res.text());
        const links = $('a').map(function() {
          if (!$(this).attr('href')) {
            return;
          }
          if ($(this).attr('href') !== '#' && $(this).attr('href').indexOf('http') > -1) {
            return $(this).attr('href');
          }
        }).get();

        return links.map(url => ({ url }))
      }
    },
    url: {
      type: GraphQLString,
      resolve(root, args, context) {
        return root.url;
      }
    },
    hostname: {
      type: GraphQLString,
      resolve(root, args, context) {
        const match = root.url.match(/^(https?\:)\/\/(([^:\/?#]*)(?:\:([0-9]+))?)([\/]{0,1}[^?#]*)(\?[^#]*|)(#.*|)$/);
        return match && match[3];
      }
    },
    title: {
      type: GraphQLString,
      resolve: async (root, args, context) => {
        const res = await fetch(root.url);
        const $ = cheerio.load(await res.text());
        return $('title').text();
      }
    }
  })
});
```

You can check out the full repo at GraphQL Hackathon Github and install the dependencies with npm install.

Then, run the code with npm start and navigate to http://localhost:3010/graphiql. There, you can try running the scrape query I showed in the exampled above.

Like this post? Please share any thoughts or feedback below.
