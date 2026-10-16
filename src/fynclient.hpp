#pragma once
#include <QTcpSocket>
#include <QAbstractSocket>
#include <QObject>
#include <QString>
#include <QByteArray>
#include <QDebug>
#include <cstdint>
#include "shared/protocol.hpp"

class FynClient : public QObject {
    Q_OBJECT
public:
    explicit FynClient(const QString& host, quint16 port, QObject *parent = nullptr);
    void sendCommand(CommandType cmd, const QString& payload);

signals:
    void connectedChanged(bool connected);
    void messageReceived(const QByteArray &data);
    void commandReceived(CommandType cmd, const QByteArray &payload);
    void errorOccurred(const QString &msg);

private slots:
    void onConnected();
    void onDisconnected();
    void onReadyRead();
    void onError(QAbstractSocket::SocketError socketError);

private:
    QTcpSocket *socket;
    QByteArray readBuffer;
};
