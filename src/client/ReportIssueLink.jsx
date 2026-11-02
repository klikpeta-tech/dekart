import Button from 'antd/es/button'
import { version } from '../../package.json'

// ReportIssueLink opens a draft without copying potentially private error text into its URL.
export default function ReportIssueLink ({ category }) {
  const params = new URLSearchParams({
    title: `Klikpeta ${category} issue`,
    body: `Klikpeta version: ${version}\nArea: ${category}\n\nWhat happened?\n\nSteps to reproduce:\n`
  })
  return (
    <Button
      href={`https://github.com/klikpeta-tech/dekart/issues/new?${params}`}
      target='_blank'
      rel='noopener noreferrer'
      type='text'
    >Report issue
    </Button>
  )
}
