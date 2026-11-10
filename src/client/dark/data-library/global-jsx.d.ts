import type { JSX as ReactJSX } from 'react'

// react-markdown 8 types read the global JSX namespace, which React 19's types no longer declare.
declare global {
  // eslint-disable-next-line @typescript-eslint/no-namespace
  namespace JSX {
    type IntrinsicElements = ReactJSX.IntrinsicElements
  }
}
