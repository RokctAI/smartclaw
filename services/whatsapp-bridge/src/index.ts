import makeWASocket, { 
    DisconnectReason, 
    useMultiFileAuthState, 
    fetchLatestBaileysVersion, 
    makeCacheableSignalKeyStore,
    downloadMediaMessage,
    proto
} from '@whiskeysockets/baileys'
import { Boom } from '@hapi/boom'
import qrcode from 'qrcode-terminal'
import pino from 'pino'
import { WebSocketServer, WebSocket } from 'ws'
import fs from 'fs'
import path from 'path'
import os from 'os'

const logger = pino({ level: 'info' })
const PORT = process.env.PORT || 3000
const AUTH_PATH = path.join(__dirname, '../auth_info')

async function connectToWhatsApp() {
    const { state, saveCreds } = await useMultiFileAuthState(AUTH_PATH)
    const { version, isLatest } = await fetchLatestBaileysVersion()
    
    logger.info(`Starting WhatsApp Bridge (v${version}, latest: ${isLatest})`)

    const sock = makeWASocket({
        version,
        logger: logger.child({ module: 'baileys' }),
        printQRInTerminal: true,
        auth: {
            creds: state.creds,
            keys: makeCacheableSignalKeyStore(state.keys, logger.child({ level: 'silent' })),
        },
        generateHighQualityLinkPreview: true,
    })

    // WebSocket Server for GoClaw Brain
    const wss = new WebSocketServer({ port: Number(PORT) })
    let goClient: WebSocket | null = null

    wss.on('connection', (ws) => {
        logger.info('GoClaw Brain connected to bridge')
        goClient = ws

        ws.on('message', async (data) => {
            try {
                const payload = JSON.parse(data.toString())
                if (payload.type === 'message') {
                    await sock.sendMessage(payload.to, { text: payload.content })
                }
            } catch (err) {
                logger.error(err, 'Failed to process brain message')
            }
        })

        ws.on('close', () => {
            logger.info('GoClaw Brain disconnected')
            goClient = null
        })
    })

    sock.ev.process(async (events) => {
        if (events['connection.update']) {
            const update = events['connection.update']
            const { connection, lastDisconnect, qr } = update
            
            if (qr) {
                logger.info('New QR Code generated. Scan to link device.')
            }

            if (connection === 'close') {
                const shouldReconnect = (lastDisconnect?.error as Boom)?.output?.statusCode !== DisconnectReason.loggedOut
                logger.info(`Connection closed due to ${lastDisconnect?.error}, reconnecting ${shouldReconnect}`)
                if (shouldReconnect) {
                    connectToWhatsApp()
                }
            } else if (connection === 'open') {
                logger.info('WhatsApp Bridge is OPEN and READY')
            }
        }

        if (events['creds.update']) {
            await saveCreds()
        }

        if (events['messages.upsert']) {
            const upsert = events['messages.upsert']
            if (upsert.type === 'notify') {
                for (const msg of upsert.messages) {
                    if (!msg.key.fromMe && msg.message) {
                        const from = msg.key.remoteJid
                        const pushName = msg.pushName || 'User'
                        
                        let content = msg.message.conversation || 
                                      msg.message.extendedTextMessage?.text || 
                                      ''

                        const mediaPaths: string[] = []

                        // Handle Media Messages (Image, Video, Document, Audio)
                        const messageType = Object.keys(msg.message)[0]
                        if (['imageMessage', 'videoMessage', 'documentMessage', 'audioMessage'].includes(messageType)) {
                            try {
                                logger.info(`Downloading media: ${messageType}`)
                                const buffer = await downloadMediaMessage(
                                    msg,
                                    'buffer',
                                    {},
                                    { logger, reuploadRequest: sock.updateMediaMessage }
                                )
                                
                                // Save to temp file so GoClaw can read it from the VPS disk
                                const tempPath = path.join(os.tmpdir(), `wa_media_${Date.now()}_${msg.key.id}`)
                                fs.writeFileSync(tempPath, buffer)
                                mediaPaths.push(tempPath)
                                
                                // Auto-caption with filename if document
                                if (messageType === 'documentMessage') {
                                    content = content || msg.message.documentMessage?.fileName || '[Document]'
                                }
                            } catch (err) {
                                logger.error(err, 'Failed to download media')
                            }
                        }

                        if (content === '' && mediaPaths.length === 0) {
                            continue
                        }

                        if (goClient && goClient.readyState === WebSocket.OPEN) {
                            goClient.send(JSON.stringify({
                                type: 'message',
                                from: from,
                                chat: from,
                                from_name: pushName,
                                content: content || '[Media Message]',
                                id: msg.key.id,
                                media: mediaPaths
                            }))
                        }
                    }
                }
            }
        }
    })
}

connectToWhatsApp().catch(err => logger.error(err, 'Fatal transition'))
