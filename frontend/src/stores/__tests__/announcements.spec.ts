import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useAnnouncementStore } from '@/stores/announcements'

const mockList = vi.fn()
const mockMarkRead = vi.fn()

vi.mock('@/api', () => ({
  announcementsAPI: {
    list: (...args: any[]) => mockList(...args),
    markRead: (...args: any[]) => mockMarkRead(...args),
  },
}))

const popupAnnouncement = {
  id: 1,
  title: 'Popup announcement',
  content: 'Read this now',
  notify_mode: 'popup' as const,
  created_at: '2026-05-10T10:00:00Z',
  updated_at: '2026-05-10T10:00:00Z',
}

const silentAnnouncement = {
  id: 2,
  title: 'Silent announcement',
  content: 'Bell only',
  notify_mode: 'silent' as const,
  created_at: '2026-05-10T10:05:00Z',
  updated_at: '2026-05-10T10:05:00Z',
}

describe('useAnnouncementStore native announcements', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.useFakeTimers()
    vi.setSystemTime(new Date('2026-05-10T12:00:00Z'))
    vi.clearAllMocks()
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  it('loads visible announcements and queues unread popup announcements', async () => {
    mockList.mockResolvedValue([popupAnnouncement, silentAnnouncement])
    const store = useAnnouncementStore()

    await store.fetchAnnouncements(true)

    expect(mockList).toHaveBeenCalledWith(false)
    expect(store.announcements).toEqual([popupAnnouncement, silentAnnouncement])
    expect(store.unreadCount).toBe(2)
    expect(store.currentPopup).toEqual(popupAnnouncement)
  })

  it('marks popup announcements as read when dismissed and then shows the next popup', async () => {
    const secondPopup = { ...popupAnnouncement, id: 3, title: 'Second popup' }
    mockList.mockResolvedValue([popupAnnouncement, secondPopup])
    mockMarkRead.mockResolvedValue({ message: 'ok' })
    const store = useAnnouncementStore()

    await store.fetchAnnouncements(true)
    await store.dismissPopup()

    expect(mockMarkRead).toHaveBeenCalledWith(1)
    expect(store.announcements[0].read_at).toBe('2026-05-10T12:00:00.000Z')
    expect(store.currentPopup).toBeNull()

    await vi.advanceTimersByTimeAsync(300)

    expect(store.currentPopup).toEqual(secondPopup)
  })

  it('throttles regular refreshes while allowing force refresh for login checks', async () => {
    mockList.mockResolvedValue([silentAnnouncement])
    const store = useAnnouncementStore()

    await store.fetchAnnouncements()
    await store.fetchAnnouncements()
    await store.fetchAnnouncements(true)

    expect(mockList).toHaveBeenCalledTimes(2)
  })
})
