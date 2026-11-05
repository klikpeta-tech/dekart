import { useEffect, useState } from 'react'
import { useDispatch, useSelector } from 'react-redux'
import Modal from 'antd/es/modal'
import Collapse from 'antd/es/collapse'
import Checkbox from 'antd/es/checkbox'
import Spin from 'antd/es/spin'
import styles from './LibraryCategoriesModal.module.css'
import { getLibraryTaxonomy } from './actions/libraryGrants'

const formatCount = (count) => new Intl.NumberFormat('id-ID').format(count)

function groupCategories (taxonomy) {
  return taxonomy.groupsList
    .slice()
    .sort((a, b) => a.sortOrder - b.sortOrder)
    .map(group => ({
      ...group,
      categories: taxonomy.categoriesList
        .filter(category => category.groupId === group.id)
        .sort((a, b) => a.sortOrder - b.sortOrder)
    }))
}

// LibraryCategoriesModal picks the granted categories of an item from its whole taxonomy.
// At least one category is required.
export default function LibraryCategoriesModal ({ itemId, title, initial, okText, onOk, onCancel }) {
  const dispatch = useDispatch()
  const taxonomy = useSelector(state => state.libraryGrants.taxonomy[itemId])
  const [selected, setSelected] = useState(initial)
  useEffect(() => {
    if (!taxonomy) {
      dispatch(getLibraryTaxonomy(itemId))
    }
  }, [dispatch, itemId, taxonomy])

  return (
    <Modal
      open
      title={`Categories · ${title}`}
      okText={okText}
      okButtonProps={{ disabled: selected.length === 0, id: 'dekart-library-categories-ok' }}
      onOk={() => onOk(selected)}
      onCancel={onCancel}
      width={640}
    >
      {taxonomy
        ? (
          <>
            <div className={styles.hint}>
              {selected.length === 0 ? 'Pick at least one category.' : `${selected.length} selected`}
            </div>
            <Checkbox.Group className={styles.groups} value={selected} onChange={setSelected}>
              <Collapse
                items={groupCategories(taxonomy).map(group => ({
                  key: group.id,
                  label: group.title,
                  forceRender: true,
                  children: (
                    <div className={styles.categories}>
                      {group.categories.map(category => (
                        <Checkbox key={category.id} value={category.id}>
                          {category.title} ({formatCount(category.rowCount)})
                        </Checkbox>
                      ))}
                    </div>
                  )
                }))}
              />
            </Checkbox.Group>
          </>
          )
        : <Spin />}
    </Modal>
  )
}
