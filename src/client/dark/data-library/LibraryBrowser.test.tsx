import { useState } from 'react'
import type { ReactElement } from 'react'
import { describe, expect, it, vi } from 'vitest'
import { fireEvent, render, screen, within } from '@testing-library/react'
import { LibraryBrowser, emptyLibraryQuery } from './index'
import type { CatalogueItem, LibraryBrowserProps, LibraryQuery } from './index'

const items: CatalogueItem[] = [
  { id: 'pop', title: 'Penduduk 2020', category: 'Demografi', provider: 'BPS', coverageSummary: 'Indonesia', shape: { kind: 'areas', adminLevel: 'kabkota' }, rowCount: 514, areaLevels: ['provinsi', 'kabkota'] },
  { id: 'poi', title: 'Tempat & Usaha', category: 'Tempat', provider: 'Klikpeta', coverageSummary: 'Indonesia', shape: { kind: 'points' }, rowCount: 2100, areaLevels: ['provinsi', 'kabkota', 'kecamatan'], searchTerms: ['Apotek'] },
  { id: 'xss', title: '<img src=x onerror=alert(1)>', category: 'Demografi', provider: 'BPS', coverageSummary: '', shape: { kind: 'lines' }, rowCount: 1, areaLevels: [] }
]

// Holds the query the way a parent would, so the controlled browser can be driven like a user would.
function Harness (props: Partial<LibraryBrowserProps> & { initial?: LibraryQuery }): ReactElement {
  const [query, setQuery] = useState(props.initial ?? emptyLibraryQuery())
  return <LibraryBrowser items={items} {...props} query={query} onQueryChange={setQuery} />
}

const rowTitles = (): Array<string | null | undefined> => screen.queryAllByRole('listitem').map(li => li.querySelector('span span')?.textContent)

describe('LibraryBrowser', () => {
  it('shows skeletons while loading', () => {
    render(<Harness items={null} />)
    expect(screen.getAllByTestId('library-skeleton')).toHaveLength(3)
    expect(screen.queryByRole('group', { name: 'Categories' })).toBeNull()
  })

  it('lists items with row counts and badges', () => {
    render(<Harness addedItemIds={['poi']} addedLabel='On this map' />)
    expect(screen.getByText('Demografi · Areas · Kabupaten/Kota · 514 rows')).toBeTruthy()
    expect(screen.getByText('Tempat · Points · 2.100 rows')).toBeTruthy()
    expect(screen.getByText('On this map')).toBeTruthy()
    expect(screen.getAllByText('Available')).toHaveLength(2)
  })

  it('renders hostile text as text', () => {
    const { container } = render(<Harness />)
    expect(screen.getByText('<img src=x onerror=alert(1)>')).toBeTruthy()
    expect(container.querySelector('img')).toBeNull()
  })

  it('narrows the list by search, including hidden search terms', () => {
    render(<Harness />)
    fireEvent.change(screen.getByRole('searchbox', { name: 'Search datasets' }), { target: { value: 'apotek' } })
    expect(screen.getByText('Tempat & Usaha')).toBeTruthy()
    expect(screen.queryByText('Penduduk 2020')).toBeNull()
  })

  it('filters by category chip and shows counts', () => {
    render(<Harness />)
    const chips = within(screen.getByRole('group', { name: 'Categories' }))
    expect(chips.getByRole('button', { name: 'All (3)' }).getAttribute('aria-pressed')).toBe('true')
    fireEvent.click(chips.getByRole('button', { name: 'Demografi (2)' }))
    expect(screen.queryByText('Tempat & Usaha')).toBeNull()
    expect(chips.getByRole('button', { name: 'Demografi (2)' }).getAttribute('aria-pressed')).toBe('true')
  })

  it('filters by shape and offers only present shapes', () => {
    render(<Harness />)
    const shape = screen.getByRole('combobox', { name: 'Shape' })
    expect(within(shape).getAllByRole('option').map(o => o.textContent)).toEqual(['All shapes', 'Points', 'Lines', 'Areas'])
    fireEvent.change(shape, { target: { value: 'points' } })
    expect(rowTitles()).toEqual(['Tempat & Usaha'])
  })

  it('applies a preset area and lists a code that has no option', () => {
    render(<Harness initial={emptyLibraryQuery('3171010')} areaOptions={[{ code: '31', name: 'DKI Jakarta' }]} />)
    expect(screen.getByRole<HTMLSelectElement>('combobox', { name: 'Area' }).value).toBe('3171010')
    expect(screen.getByRole('option', { name: 'Area 3171010' })).toBeTruthy()
    expect(rowTitles()).toEqual(['Tempat & Usaha'])
    fireEvent.change(screen.getByRole('combobox', { name: 'Area' }), { target: { value: '31' } })
    expect(rowTitles()).toEqual(['Penduduk 2020', 'Tempat & Usaha'])
  })

  it('shows an empty state that clears filters', () => {
    render(<Harness />)
    fireEvent.change(screen.getByRole('searchbox', { name: 'Search datasets' }), { target: { value: 'nothing here' } })
    expect(screen.getByText('No datasets match your filters.')).toBeTruthy()
    fireEvent.click(screen.getByRole('button', { name: 'Clear filters' }))
    expect(screen.getAllByRole('listitem')).toHaveLength(3)
  })

  it('shows a different empty state for an empty catalogue', () => {
    render(<Harness items={[]} />)
    expect(screen.getByText('No datasets are available yet.')).toBeTruthy()
    expect(screen.queryByRole('button', { name: 'Clear filters' })).toBeNull()
  })

  it('reports row selection and back, and rows are inert without a handler', () => {
    const onSelectItem = vi.fn()
    const onBack = vi.fn()
    const { unmount } = render(<Harness onSelectItem={onSelectItem} onBack={onBack} />)
    fireEvent.click(screen.getByRole('button', { name: /Tempat & Usaha/ }))
    fireEvent.click(screen.getByRole('button', { name: 'Back' }))
    expect(onSelectItem).toHaveBeenCalledWith('poi')
    expect(onBack).toHaveBeenCalledTimes(1)
    unmount()
    render(<Harness />)
    expect(screen.queryByRole('button', { name: /Tempat & Usaha/ })).toBeNull()
    expect(screen.queryByRole('button', { name: 'Back' })).toBeNull()
  })
})
