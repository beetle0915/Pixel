import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

import { describe, expect, it } from 'vitest'

const layoutDir = dirname(fileURLToPath(import.meta.url))
const sidebarSource = readFileSync(resolve(layoutDir, '../AppSidebar.vue'), 'utf8')
const headerSource = readFileSync(resolve(layoutDir, '../AppHeader.vue'), 'utf8')
const appSource = readFileSync(resolve(layoutDir, '../../../App.vue'), 'utf8')
const routerSource = readFileSync(resolve(layoutDir, '../../../router/index.ts'), 'utf8')

describe('native announcement feature exposure', () => {
  it('keeps the admin announcement management route and sidebar entry enabled', () => {
    expect(routerSource).toContain("path: '/admin/announcements'")
    expect(routerSource).toContain("component: () => import('@/views/admin/AnnouncementsView.vue')")
    expect(sidebarSource).toContain("path: '/admin/announcements'")
    expect(sidebarSource).toContain("t('nav.announcements')")
  })

  it('mounts announcement bell and popup for authenticated users', () => {
    expect(headerSource).toContain('<AnnouncementBell v-if="user" />')
    expect(headerSource).toContain("import AnnouncementBell from '@/components/common/AnnouncementBell.vue'")
    expect(appSource).toContain('<AnnouncementPopup />')
    expect(appSource).toContain("import AnnouncementPopup from '@/components/common/AnnouncementPopup.vue'")
  })

  it('fetches announcements on login, route change, and tab visibility restore', () => {
    expect(appSource).toContain('announcementStore.fetchAnnouncements(true)')
    expect(appSource).toContain('announcementStore.fetchAnnouncements()')
    expect(appSource).toContain("document.addEventListener('visibilitychange', onVisibilityChange)")
    expect(appSource).toContain('router.afterEach')
  })
})
