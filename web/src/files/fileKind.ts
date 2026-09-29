import { File, FileArchive, FileCode, FileCog, FileImage, FileMusic, FilePlay, FileText, type LucideIcon } from 'lucide-react'

export type FileKind = 'image' | 'video' | 'audio' | 'document' | 'archive' | 'program' | 'code' | 'file'

// Extension → famille de fichier, pour choisir l'icône et le libellé de la tuile.
const EXTENSIONS: Record<Exclude<FileKind, 'file'>, string[]> = {
  image: ['jpg', 'jpeg', 'png', 'gif', 'bmp', 'webp', 'svg', 'ico', 'heic', 'avif', 'tif', 'tiff', 'raw', 'psd'],
  video: ['mp4', 'mkv', 'avi', 'mov', 'wmv', 'webm', 'flv', 'm4v', 'mpg', 'mpeg', 'ts'],
  audio: ['mp3', 'wav', 'flac', 'ogg', 'm4a', 'aac', 'wma', 'opus'],
  document: ['pdf', 'txt', 'md', 'doc', 'docx', 'odt', 'rtf', 'xls', 'xlsx', 'ods', 'csv', 'ppt', 'pptx', 'odp', 'epub', 'log'],
  archive: ['zip', 'rar', '7z', 'tar', 'gz', 'bz2', 'xz', 'zst', 'iso', 'img'],
  program: ['exe', 'msi', 'bat', 'cmd', 'lnk', 'url', 'appimage', 'deb', 'rpm', 'sh', 'desktop', 'dll', 'so'],
  code: ['js', 'ts', 'tsx', 'jsx', 'py', 'go', 'rs', 'c', 'cpp', 'h', 'cs', 'java', 'html', 'css', 'json', 'xml', 'yml', 'yaml', 'toml', 'ini', 'cfg', 'ps1'],
}

const BY_EXTENSION = new Map<string, FileKind>(
  Object.entries(EXTENSIONS).flatMap(([kind, exts]) => exts.map((e) => [e, kind as FileKind])),
)

export const FILE_ICONS: Record<FileKind, LucideIcon> = {
  image: FileImage,
  video: FilePlay,
  audio: FileMusic,
  document: FileText,
  archive: FileArchive,
  program: FileCog,
  code: FileCode,
  file: File,
}

export function fileKind(name: string): FileKind {
  const dot = name.lastIndexOf('.')
  if (dot <= 0) return 'file'
  return BY_EXTENSION.get(name.slice(dot + 1).toLowerCase()) ?? 'file'
}
