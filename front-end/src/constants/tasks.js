export const TASK_TYPES = [
  {
    value: 'report',
    label: 'Report post',
    icon: 'ti-flag',
    description: 'Report a post URL with selected accounts',
    needsUrl: true,
    needsCompose: false,
    needsReply: false,
  },
  {
    value: 'reply',
    label: 'Reply / comment',
    icon: 'ti-message',
    description: 'Comment on a post URL with selected accounts',
    needsUrl: false,
    needsCompose: false,
    needsReply: true,
  },
  {
    value: 'post',
    label: 'New post',
    icon: 'ti-pencil-plus',
    description: 'Publish a new article or update from selected accounts',
    needsUrl: false,
    needsCompose: true,
    needsReply: false,
  },
  {
    value: 'browse',
    label: 'Browse feed',
    icon: 'ti-player-play',
    description: 'Browse the feed with human-like scrolling',
    needsUrl: false,
    needsCompose: false,
    needsReply: false,
  },
  {
    value: 'login_test',
    label: 'Login test',
    icon: 'ti-shield-check',
    description: 'Verify saved session / credentials',
    needsUrl: false,
    needsCompose: false,
    needsReply: false,
  },
]

export const TASK_STATUSES = [
  { value: 'queued', label: 'Queued' },
  { value: 'running', label: 'Running' },
  { value: 'completed', label: 'Completed' },
  { value: 'failed', label: 'Failed' },
  { value: 'cancelled', label: 'Cancelled' },
]

const DEFAULT_COMPOSE = {
  textLabel: 'Post text',
  textPlaceholder: 'Write your post…',
  maxText: 5000,
  needsHeadline: false,
  headlineLabel: 'Headline',
  needsMedia: false,
  mediaLabel: 'Media URL',
  mediaPlaceholder: 'https://…',
  allowsLink: true,
  tip: 'Write the content you want published on this platform.',
}

const POST_COMPOSE_BY_PLATFORM = {
  facebook: {
    ...DEFAULT_COMPOSE,
    textLabel: 'Post text',
    textPlaceholder: 'What’s on your mind?',
    maxText: 63206,
    tip: 'Facebook needs the post body. You can optionally attach a link.',
  },
  twitter: {
    ...DEFAULT_COMPOSE,
    textLabel: 'Tweet text',
    textPlaceholder: 'What’s happening?',
    maxText: 280,
    tip: 'Keep tweets under 280 characters. Links count toward the limit.',
  },
  linkedin: {
    ...DEFAULT_COMPOSE,
    textLabel: 'Post body',
    textPlaceholder: 'Share an update with your network…',
    maxText: 3000,
    needsHeadline: true,
    headlineLabel: 'Headline',
    tip: 'LinkedIn posts work best with a short headline plus body text.',
  },
  instagram: {
    ...DEFAULT_COMPOSE,
    textLabel: 'Caption',
    textPlaceholder: 'Write a caption…',
    maxText: 2200,
    needsMedia: true,
    mediaLabel: 'Image / video URL',
    mediaPlaceholder: 'https://cdn.example.com/photo.jpg',
    allowsLink: false,
    tip: 'Instagram requires media. Paste a public image or video URL for now.',
  },
  tiktok: {
    ...DEFAULT_COMPOSE,
    textLabel: 'Caption',
    textPlaceholder: 'Add a caption…',
    maxText: 2200,
    needsMedia: true,
    mediaLabel: 'Video URL',
    mediaPlaceholder: 'https://cdn.example.com/video.mp4',
    allowsLink: false,
    tip: 'TikTok requires a video. Paste a public video URL for now.',
  },
  youtube: {
    ...DEFAULT_COMPOSE,
    textLabel: 'Description',
    textPlaceholder: 'Tell viewers about your video…',
    maxText: 5000,
    needsHeadline: true,
    headlineLabel: 'Video title',
    needsMedia: true,
    mediaLabel: 'Video URL',
    mediaPlaceholder: 'https://cdn.example.com/video.mp4',
    allowsLink: false,
    tip: 'YouTube needs a title, description, and video URL.',
  },
}

export function taskTypeMeta(value) {
  return (
    TASK_TYPES.find((item) => item.value === value) || {
      value,
      label: value,
      icon: 'ti-list-check',
      description: '',
      needsUrl: false,
      needsCompose: false,
      needsReply: false,
    }
  )
}

export function postComposeSpec(platform) {
  return POST_COMPOSE_BY_PLATFORM[platform] || DEFAULT_COMPOSE
}

export function taskContentPreview(task) {
  const content = task?.content
  if (!content || typeof content !== 'object') return ''
  const headline = String(content.headline || '').trim()
  const text = String(content.text || '').trim()
  if (headline && text) return `${headline} — ${text}`
  return headline || text
}

export function taskStatusLabel(value) {
  return TASK_STATUSES.find((item) => item.value === value)?.label || value
}

export function taskProgress(task) {
  const total = Number(task?.accountCount || 0)
  const done = Number(task?.doneCount || 0)
  if (!total) return 0
  return Math.round((done / total) * 100)
}
