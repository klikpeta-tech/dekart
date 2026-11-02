import { useState } from 'react'
import type { ReactElement } from 'react'
import { describe, expect, it, vi } from 'vitest'
import { fireEvent, render, screen, waitFor, within } from '@testing-library/react'
import { AreaPicker, CategoryPicker, LibraryDatasetStatus, LibraryItemDetail } from './index'
import type { AreaNode, ItemDetail, LibraryDatasetStatusProps, Taxonomy } from './index'

const item: ItemDetail = {
  id: 'poi',
  title: 'Tempat Usaha',
  category: 'Places',
  provider: 'Klikpeta',
  coverageSummary: 'Seluruh Indonesia',
  shape: { kind: 'points' },
  rowCount: 2100000,
  areaLevels: ['provinsi', 'kabkota'],
  description: 'Paragraf pertama.\n\nParagraf <b>kedua</b>.',
  attribution: 'Sumber: OpenStreetMap',
  sourceUpdatedAt: Date.UTC(2026, 9, 1) / 1000,
  cadence: 'Kuartalan',
  columns: ['nama', 'alamat'],
  maxRows: 1000000,
  hasTaxonomy: true
}

const taxonomy: Taxonomy = {
  groups: [{ id: 'g1', title: 'Makanan' }, { id: 'g2', title: 'Kesehatan' }],
  categories: [
    { id: 'kafe', groupId: 'g1', title: 'Kafe' },
    { id: 'kopi', groupId: 'g1', title: 'Kedai kopi' },
    { id: 'apotek', groupId: 'g2', title: 'Apotek' }
  ],
  brands: [
    { id: 'starbucks', categoryId: 'kopi', title: 'Starbucks' },
    { id: 'kimia_farma', categoryId: 'apotek', title: 'Kimia Farma' }
  ]
}

describe('LibraryItemDetail', () => {
  it('shows description, facts, columns and the credit line as text', () => {
    const { container } = render(<LibraryItemDetail item={item} />)
    expect(screen.getByRole('heading', { name: 'Tempat Usaha' })).toBeTruthy()
    expect(screen.getByText('Paragraf <b>kedua</b>.')).toBeTruthy()
    expect(container.querySelector('b')).toBeNull()
    expect(screen.getByText('2.100.000')).toBeTruthy()
    expect(screen.getByText('1 Oktober 2026')).toBeTruthy()
    expect(screen.getByText('Kuartalan')).toBeTruthy()
    expect(screen.getByText('nama')).toBeTruthy()
    expect(screen.getByText('Sumber: OpenStreetMap')).toBeTruthy()
  })

  it('leaves out facts with no value', () => {
    render(<LibraryItemDetail item={{ ...item, cadence: '', sourceUpdatedAt: 0, attribution: '', columns: [] }} />)
    expect(screen.queryByText('Refresh cadence')).toBeNull()
    expect(screen.queryByText('Source updated')).toBeNull()
    expect(screen.queryByText('Columns')).toBeNull()
  })

  it('drives the primary action, the preview and the back button', () => {
    const onClick = vi.fn()
    const onPreviewSample = vi.fn()
    const onBack = vi.fn()
    render(
      <LibraryItemDetail
        item={item}
        primary={{ label: 'Add 2.100 places to map', disabled: false, onClick }}
        onPreviewSample={onPreviewSample}
        onBack={onBack}
        sample={{ columns: ['nama'], rows: [['<i>Kopi</i>']] }}
        issues={[{ code: 'empty', message: 'No rows match this selection.' }]}
        notice='Access ended'
      />
    )
    fireEvent.click(screen.getByRole('button', { name: 'Add 2.100 places to map' }))
    fireEvent.click(screen.getByRole('button', { name: 'Preview sample' }))
    fireEvent.click(screen.getByRole('button', { name: 'Back' }))
    expect([onClick, onPreviewSample, onBack].map(fn => fn.mock.calls.length)).toEqual([1, 1, 1])
    expect(within(screen.getByRole('table', { name: 'Sample rows' })).getByText('<i>Kopi</i>')).toBeTruthy()
    expect(screen.getByText('No rows match this selection.')).toBeTruthy()
    expect(screen.getByText('Access ended')).toBeTruthy()
  })

  it('does not call a disabled primary action', () => {
    const onClick = vi.fn()
    render(<LibraryItemDetail item={item} primary={{ label: 'Add to map', disabled: true, onClick }} />)
    fireEvent.click(screen.getByRole('button', { name: 'Add to map' }))
    expect(onClick).not.toHaveBeenCalled()
  })
})

const areas: Record<string, AreaNode[]> = {
  '': [{ code: '31', name: 'DKI Jakarta', rowCount: 900 }, { code: '32', name: 'Jawa Barat', rowCount: 2100 }],
  32: [{ code: '3273', name: 'Kota Bandung', rowCount: 1500 }],
  3273: [{ code: '3273010', name: 'Sukasari' }]
}

function Harness ({ initial = '', levels = ['provinsi', 'kabkota', 'kecamatan'], onName }: { initial?: string, levels?: string[], onName?: (name: string) => void }): ReactElement {
  const [value, setValue] = useState(initial)
  return <AreaPicker levels={levels} value={value} onChange={setValue} loadAreas={async parent => areas[parent] ?? []} onNameChange={onName} />
}

describe('AreaPicker', () => {
  it('walks down from province to sub-district and shows counts', async () => {
    render(<Harness />)
    const province = screen.getByRole('combobox', { name: 'Provinsi' })
    await waitFor(() => expect(within(province).getByText('Jawa Barat (2.100)')).toBeTruthy())
    expect(screen.getByRole<HTMLSelectElement>('combobox', { name: 'Kabupaten/Kota' }).disabled).toBe(true)

    fireEvent.change(province, { target: { value: '32' } })
    await waitFor(() => expect(screen.getByText('Kota Bandung (1.500)')).toBeTruthy())
    fireEvent.change(screen.getByRole('combobox', { name: 'Kabupaten/Kota' }), { target: { value: '3273' } })
    await waitFor(() => expect(screen.getByText('Sukasari')).toBeTruthy())
    fireEvent.change(screen.getByRole('combobox', { name: 'Kecamatan' }), { target: { value: '3273010' } })
    expect(screen.getByRole<HTMLSelectElement>('combobox', { name: 'Kecamatan' }).value).toBe('3273010')
  })

  it('picking "All" at a level goes back to its parent', async () => {
    const onChange = vi.fn()
    render(<AreaPicker levels={['provinsi', 'kabkota']} value='3273' onChange={onChange} loadAreas={async parent => areas[parent] ?? []} />)
    await waitFor(() => expect(screen.getByText('Kota Bandung (1.500)')).toBeTruthy())
    fireEvent.change(screen.getByRole('combobox', { name: 'Kabupaten/Kota' }), { target: { value: '' } })
    expect(onChange).toHaveBeenLastCalledWith('32')
    fireEvent.change(screen.getByRole('combobox', { name: 'Provinsi' }), { target: { value: '' } })
    expect(onChange).toHaveBeenLastCalledWith('')
  })

  it('shows only the levels the item supports and reports the picked name', async () => {
    const onName = vi.fn()
    render(<Harness initial='3273' levels={['provinsi', 'kabkota']} onName={onName} />)
    expect(screen.queryByRole('combobox', { name: 'Kecamatan' })).toBeNull()
    await waitFor(() => expect(onName).toHaveBeenLastCalledWith('Kota Bandung'))
  })

  it('renders nothing for an item that cannot be narrowed', () => {
    const { container } = render(<Harness levels={[]} />)
    expect(container.firstChild).toBeNull()
  })

  it('treats a malformed value as the whole coverage and reports a failed load', async () => {
    render(<AreaPicker levels={['provinsi']} value='99x' onChange={() => {}} loadAreas={async () => { throw new Error('down') }} />)
    await waitFor(() => expect(screen.getByRole('alert').textContent).toBe('Could not load Provinsi.'))
    expect(screen.getByRole<HTMLSelectElement>('combobox', { name: 'Provinsi' }).value).toBe('')
  })
})

describe('CategoryPicker', () => {
  function Picking ({ onChange }: { onChange?: (next: { categoryIds: string[], brandIds: string[] }) => void }): ReactElement {
    const [state, setState] = useState<{ categoryIds: string[], brandIds: string[] }>({ categoryIds: [], brandIds: [] })
    return (
      <CategoryPicker
        taxonomy={taxonomy}
        {...state}
        categoryCounts={{ kafe: 1200 }}
        onChange={next => { setState(next); onChange?.(next) }}
      />
    )
  }

  it('asks for a category until one is picked, and shows counts', () => {
    render(<Picking />)
    expect(screen.getByText('Pick at least one category.')).toBeTruthy()
    expect(screen.getByLabelText('Kafe (1.200)')).toBeTruthy()
    fireEvent.click(screen.getByLabelText('Kedai kopi'))
    expect(screen.getByText('1 selected')).toBeTruthy()
    expect(screen.getByText('Makanan (1 selected)')).toBeTruthy()
  })

  it('offers brands of the picked categories only and drops them with their category', () => {
    const onChange = vi.fn()
    render(<Picking onChange={onChange} />)
    expect(screen.queryByRole('group', { name: 'Brands' })).toBeNull()
    fireEvent.click(screen.getByLabelText('Kedai kopi'))
    const brands = screen.getByRole('group', { name: 'Brands' })
    expect(within(brands).queryByText('Kimia Farma')).toBeNull()
    fireEvent.click(within(brands).getByRole('button', { name: 'Starbucks' }))
    expect(onChange).toHaveBeenLastCalledWith({ categoryIds: ['kopi'], brandIds: ['starbucks'] })
    fireEvent.click(screen.getByLabelText('Kedai kopi'))
    expect(onChange).toHaveBeenLastCalledWith({ categoryIds: [], brandIds: [] })
    expect(screen.queryByRole('group', { name: 'Brands' })).toBeNull()
  })

  it('renders hostile titles as text', () => {
    const { container } = render(
      <CategoryPicker
        taxonomy={{ groups: [{ id: 'g', title: '<img src=x>' }], categories: [{ id: 'c', groupId: 'g', title: '<script>alert(1)</script>' }], brands: [] }}
        categoryIds={[]}
        brandIds={[]}
        onChange={() => {}}
      />
    )
    expect(screen.getByText('<script>alert(1)</script>')).toBeTruthy()
    expect(container.querySelector('img, script')).toBeNull()
  })
})

describe('LibraryDatasetStatus', () => {
  const base: LibraryDatasetStatusProps = {
    title: 'Tempat Usaha · Kafe · Kota Bandung',
    state: 'ready',
    canChange: true,
    canRefresh: true,
    hasCategories: true,
    onChangeCategory: vi.fn(),
    onChangeArea: vi.fn(),
    onRefresh: vi.fn(),
    onRetry: vi.fn(),
    onOpenLibrary: vi.fn()
  }

  it('shows ready with every action', () => {
    render(<LibraryDatasetStatus {...base} />)
    expect(screen.getByRole('status').textContent).toBe('Ready')
    for (const name of ['Change category', 'Change area', 'Refresh', 'Open in Library']) {
      expect(screen.getByRole<HTMLButtonElement>('button', { name }).disabled).toBe(false)
    }
  })

  it('shows loading and cannot be refreshed meanwhile', () => {
    render(<LibraryDatasetStatus {...base} state='loading' />)
    expect(screen.getByRole('status').textContent).toBe('Loading…')
    expect(screen.getByRole<HTMLButtonElement>('button', { name: 'Refresh' }).disabled).toBe(true)
  })

  it('shows a failure as text with a retry', () => {
    const onRetry = vi.fn()
    render(<LibraryDatasetStatus {...base} state='failed' error='<b>boom</b>' onRetry={onRetry} />)
    expect(screen.getByRole('status').textContent).toBe('Failed: <b>boom</b>Retry')
    fireEvent.click(screen.getByRole('button', { name: 'Retry' }))
    expect(onRetry).toHaveBeenCalledTimes(1)
  })

  it('shows access ended with its date and disables refresh and changes, but not opening the item', () => {
    render(<LibraryDatasetStatus {...base} state='ended' accessEndedAt={Date.UTC(2026, 10, 2) / 1000} />)
    expect(screen.getByRole('status').textContent).toBe('Access ended 2 November 2026. Your Klikpeta contact can extend it.')
    for (const name of ['Change category', 'Change area', 'Refresh']) {
      expect(screen.getByRole<HTMLButtonElement>('button', { name }).disabled).toBe(true)
    }
    expect(screen.getByRole<HTMLButtonElement>('button', { name: 'Open in Library' }).disabled).toBe(false)
  })

  it('leaves out the date when unknown, and hides actions the user may not take', () => {
    render(<LibraryDatasetStatus {...base} state='ended' canChange={false} canRefresh={false} onOpenLibrary={undefined} />)
    expect(screen.getByRole('status').textContent).toBe('Access ended. Your Klikpeta contact can extend it.')
    expect(screen.queryAllByRole('button')).toHaveLength(0)
  })

  it('hides the category button for items without categories', () => {
    render(<LibraryDatasetStatus {...base} hasCategories={false} />)
    expect(screen.queryByRole('button', { name: 'Change category' })).toBeNull()
  })
})
