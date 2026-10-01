/// <reference types="vite/client" />

declare module 'html-to-pdfmake' {
  import type { Content } from 'pdfmake/interfaces'
  export default function htmlToPdfmake(html: string): Content
}
