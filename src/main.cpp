#include <QApplication>
#include <QDebug>
#include <QTimer>
#include <QDataStream>
#include <QTextEdit>
#include <QLabel>
#include <QPushButton>
#include "fynclient.hpp"
#include "mainwindow.hpp"
#include "shared/protocol.hpp"

int main(int argc, char *argv[])
{
    QApplication app(argc, argv);
    qDebug() << "FynChat Desktop starting...";

    FynClient client("127.0.0.1", 8080);

    MainWindow window(&client);
    QObject::connect(&window, &MainWindow::authSucceeded, [](){
        qDebug() << "Authentication succeeded";
    });
    window.show();

    // Heartbeat timer
    QTimer heartbeat;
    heartbeat.setInterval(30000);
    QObject::connect(&heartbeat, &QTimer::timeout, [&](){
        client.sendCommand(CommandType::HeartBeat, "");
    });

    QObject::connect(&client, &FynClient::connectedChanged, [&](bool connected){
        if (connected) {
            qDebug() << "Connected, fetching history";
            heartbeat.start();
            client.sendCommand(CommandType::FetchHistory, "");
        } else {
            heartbeat.stop();
        }
    });

    // Handle parsed commands from client
    QObject::connect(&client, &FynClient::commandReceived, [&](CommandType cmd, const QByteArray &payload){
        QString payloadStr = QString::fromUtf8(payload);

        switch (cmd) {
        case CommandType::FetchHistory:
            window.addChatMessage("[History] " + payloadStr, true);
            break;
        case CommandType::ChatMessage:
            window.addChatMessage(payloadStr, true);
            break;
        case CommandType::TypingStatus:
            window.headerLabel()->setText(payloadStr == "1" ? "FynChat - typing..." : "FynChat");
            break;
        case CommandType::Login:
        case CommandType::Register:
            // Auth handled by MainWindow::showAuthDialog
            qDebug() << "Auth response received:" << payloadStr;
            break;
        default:
            qDebug() << "Received cmd" << static_cast<uint16_t>(cmd) << "payload:" << payloadStr;
        }
    });

    // Propagate client errors to UI
    QObject::connect(&client, &FynClient::errorOccurred, [&](const QString &msg){
        qWarning() << "Client error:" << msg;
    });

    // Typing indicator
    QObject::connect(window.inputEdit(), &QTextEdit::textChanged, [&](){
        static bool wasEmpty = true;
        bool isEmpty = window.inputEdit()->toPlainText().isEmpty();
        if (wasEmpty != isEmpty) {
            client.sendCommand(CommandType::TypingStatus, isEmpty ? "0" : "1");
            wasEmpty = isEmpty;
        }
    });

    // Send on button click
    if (auto *sendBtn = window.findChild<QPushButton*>("sendButton")) {
        QObject::connect(sendBtn, &QPushButton::clicked, [&](){
            QString text = window.inputEdit()->toPlainText().trimmed();
            if (!text.isEmpty()) {
                client.sendCommand(CommandType::ChatMessage, text);
                window.addChatMessage(text, false);
                window.inputEdit()->clear();
                client.sendCommand(CommandType::TypingStatus, "0");
            }
        });
    }

    return app.exec();
}
