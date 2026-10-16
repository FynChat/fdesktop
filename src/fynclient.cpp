#include "fynclient.hpp"
#include <QDataStream>

FynClient::FynClient(const QString& host, quint16 port, QObject *parent)
    : QObject(parent), socket(new QTcpSocket(this))
{
    connect(socket, &QTcpSocket::connected, this, &FynClient::onConnected);
    connect(socket, &QTcpSocket::disconnected, this, &FynClient::onDisconnected);
    connect(socket, &QTcpSocket::readyRead, this, &FynClient::onReadyRead);
    connect(socket, QOverload<QAbstractSocket::SocketError>::of(&QTcpSocket::errorOccurred),
            this, &FynClient::onError);

    socket->connectToHost(host, port);
}

void FynClient::sendCommand(CommandType cmd, const QString& payload)
{
    if (socket->state() != QAbstractSocket::ConnectedState) {
        qWarning() << "[Client] Socket not connected, cannot send command";
        emit errorOccurred("Socket not connected");
        return;
    }
    QByteArray payloadBytes = payload.toUtf8();
    uint32_t headerSize = sizeof(FcpHeader);
    if (payloadBytes.size() > static_cast<qint64>(UINT32_MAX - headerSize)) {
        qWarning() << "[Client] Payload too large";
        emit errorOccurred("Payload too large");
        return;
    }
    uint32_t totalLength = headerSize + static_cast<uint32_t>(payloadBytes.size());

    QByteArray packet;
    packet.reserve(totalLength);
    QDataStream out(&packet, QIODevice::WriteOnly);
    out.setByteOrder(QDataStream::LittleEndian);
    out << totalLength;
    out << static_cast<uint16_t>(cmd);

    packet.append(payloadBytes);

    qint64 written = 0;
    while (written < packet.size()) {
        qint64 n = socket->write(packet.constData() + written, packet.size() - written);
        if (n <= 0) {
            qWarning() << "[Client] Write error or socket not writable";
            break;
        }
        written += n;
    }
    if (written != packet.size()) {
        qWarning() << "[Client] Partial write:" << written << "/" << packet.size();
    } else {
        socket->flush();
    }
    qDebug() << "[Client] Sent command ID:" << static_cast<uint16_t>(cmd) << "Payload:" << payload;
}

void FynClient::onConnected()
{
    qDebug() << "[Client] Connected to FynChat server successfully!";
    emit connectedChanged(true);
}

void FynClient::onDisconnected()
{
    qDebug() << "[Client] Disconnected";
    readBuffer.clear();
    emit connectedChanged(false);
}

void FynClient::onReadyRead()
{
    readBuffer.append(socket->readAll());

    constexpr uint32_t HEADER_SIZE = sizeof(FcpHeader);
    constexpr uint32_t MAX_PACKET = 4 * 1024 * 1024;

    while (readBuffer.size() >= HEADER_SIZE) {
        QDataStream ds(readBuffer);
        ds.setByteOrder(QDataStream::LittleEndian);
        uint32_t packetLength = 0;
        uint16_t commandId = 0;
        ds >> packetLength >> commandId;

        if (packetLength < HEADER_SIZE || packetLength > MAX_PACKET) {
            qWarning() << "[Client] Invalid packet length:" << packetLength << "discarding header";
            readBuffer.remove(0, HEADER_SIZE);
            continue;
        }
        if (readBuffer.size() < packetLength) {
            break;
        }
        QByteArray packet = readBuffer.left(packetLength);
        readBuffer.remove(0, packetLength);
        qDebug() << "[Client] Received packet size:" << packet.size() << "cmd:" << commandId;
        emit messageReceived(packet);

        QByteArray payload = packet.mid(HEADER_SIZE);
        emit commandReceived(static_cast<CommandType>(commandId), payload);
    }
}

void FynClient::onError(QAbstractSocket::SocketError)
{
    QString msg = socket->errorString();
    qDebug() << "[Client] Socket error:" << msg;
    emit errorOccurred(msg);
}
