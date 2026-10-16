#include "mainwindow.hpp"
#include "logindialog.hpp"
#include "fynclient.hpp"
#include "shared/protocol.hpp"
#include <QVBoxLayout>
#include <QHBoxLayout>
#include <QListWidget>
#include <QTextEdit>
#include <QPushButton>
#include <QLabel>
#include <QLineEdit>
#include <QSplitter>
#include <QPalette>
#include <QMenuBar>
#include <QMenu>
#include <QAction>
#include <QMessageBox>
#include <QFileDialog>
#include <QFile>
#include <QByteArray>
#include <QBuffer>
#include <QPixmap>

MainWindow::MainWindow(FynClient *client, QWidget *parent)
    : QMainWindow(parent), m_client(client), m_loggedIn(false)
{
    setWindowTitle("FynChat Desktop v0.1.0");
    resize(1000, 700);
    setupUI();
    applyTheme();
    setupMenu();

    if (!m_loggedIn) {
        showAuthDialog();
    }
}

void MainWindow::addChatMessage(const QString &text, bool incoming)
{
    QListWidgetItem *item = new QListWidgetItem(text);
    item->setTextAlignment(incoming ? Qt::AlignLeft : Qt::AlignRight);
    chatHistory->addItem(item);
    chatHistory->scrollToBottom();
}

void MainWindow::setupUI()
{
    QWidget *central = new QWidget(this);
    QHBoxLayout *mainLayout = new QHBoxLayout(central);
    mainLayout->setContentsMargins(0,0,0,0);
    mainLayout->setSpacing(0);

    splitter = new QSplitter(Qt::Horizontal, this);

    // Sidebar
    sidebar = new QWidget();
    QVBoxLayout *sidebarLayout = new QVBoxLayout(sidebar);
    sidebarLayout->setContentsMargins(12,12,12,12);
    sidebarLayout->setSpacing(8);

    QLabel *sidebarTitle = new QLabel("Chats");
    sidebarTitle->setStyleSheet("font-weight:600; font-size:16px;");
    searchBar = new QLineEdit();
    searchBar->setPlaceholderText("Search chats...");
    searchBar->setClearButtonEnabled(true);

    chatList = new QListWidget();
    chatList->setMaximumHeight(400);
    chatList->addItem("General");
    chatList->addItem("Direct Messages");
    chatList->addItem("Support");

    sidebarLayout->addWidget(sidebarTitle);
    sidebarLayout->addWidget(searchBar);
    sidebarLayout->addWidget(chatList);
    sidebarLayout->addStretch();

    // Chat pane
    chatPane = new QWidget();
    QVBoxLayout *chatLayout = new QVBoxLayout(chatPane);
    chatLayout->setContentsMargins(0,0,0,0);
    chatLayout->setSpacing(0);

    // Header
    QWidget *headerWidget = new QWidget();
    headerWidget->setStyleSheet("background:#1e1e2e; padding:12px;");
    QHBoxLayout *headerLayout = new QHBoxLayout(headerWidget);
    avatarLabel = new QLabel();
    avatarLabel->setFixedSize(32,32);
    avatarLabel->setStyleSheet("background:#313244; border-radius:16px;");
    avatarLabel->setPixmap(QPixmap(":/resources/avatar.png").scaled(32,32,Qt::KeepAspectRatio,Qt::SmoothTransformation));
    chatHeader = new QLabel("FynChat");
    chatHeader->setStyleSheet("font-weight:600; font-size:16px;");
    QLabel *statusLabel = new QLabel("● Online");
    statusLabel->setStyleSheet("color:#a6e3a1;");
    headerLayout->addWidget(avatarLabel);
    headerLayout->addWidget(chatHeader);
    headerLayout->addStretch();
    headerLayout->addWidget(statusLabel);

    // History
    chatHistory = new QListWidget();
    chatHistory->setStyleSheet(
        "QListWidget { background:#11111b; border:none; }"
        "QListWidget::item { padding:8px; border-bottom:1px solid #313244; }"
    );

    // Input
    QWidget *inputWidget = new QWidget();
    inputWidget->setStyleSheet("background:#1e1e2e; padding:8px;");
    QHBoxLayout *inputLayout = new QHBoxLayout(inputWidget);

    emojiButton = new QPushButton("😊");
    attachButton = new QPushButton("📎");
    connect(attachButton, &QPushButton::clicked, this, [this](){
        QString fileName = QFileDialog::getOpenFileName(this, "Select Avatar", "", "Images (*.png *.jpg *.jpeg)");
        if (!fileName.isEmpty() && m_client) {
            QFile file(fileName);
            const qint64 MAX_SIZE = 5 * 1024 * 1024; // 5 MB
            if (!file.exists() || file.size() > MAX_SIZE) {
                QMessageBox::warning(this, "File too large", "Avatar must be smaller than 5 MB");
                return;
            }
            if (file.open(QIODevice::ReadOnly)) {
                QByteArray data = file.readAll();
                QByteArray base64 = data.toBase64();
                m_client->sendCommand(CommandType::AvatarUpload, base64);
                QPixmap pm(fileName);
                if (!pm.isNull()) avatarLabel->setPixmap(pm.scaled(32,32,Qt::KeepAspectRatio,Qt::SmoothTransformation));
            }
        }
    });
    inputArea = new QTextEdit();
    inputArea->setPlaceholderText("Type a message...");
    inputArea->setMaximumHeight(100);
    inputArea->setStyleSheet("background:#11111b; border:1px solid #313244; border-radius:6px;");
    sendButton = new QPushButton("Send");
    sendButton->setObjectName("sendButton");
    sendButton->setStyleSheet(
        "QPushButton { background:#89b4fa; color:#1e1e2e; border:none; border-radius:6px; padding:6px 14px; font-weight:600; }"
        "QPushButton:hover { background:#74c7ec; }"
    );

    inputLayout->addWidget(emojiButton);
    inputLayout->addWidget(attachButton);
    inputLayout->addWidget(inputArea, 1);
    inputLayout->addWidget(sendButton);

    chatLayout->addWidget(headerWidget);
    chatLayout->addWidget(chatHistory, 1);
    chatLayout->addWidget(inputWidget);

    splitter->addWidget(sidebar);
    splitter->addWidget(chatPane);
    splitter->setSizes({260, 740});

    mainLayout->addWidget(splitter);
    setCentralWidget(central);

    styleWidgets();
}

void MainWindow::styleWidgets()
{
    sidebar->setStyleSheet("background:#11111b; color:#cdd6f4;");
    chatList->setStyleSheet(
        "QListWidget { background:#11111b; border:none; }"
        "QListWidget::item { padding:8px; border-radius:6px; }"
        "QListWidget::item:selected { background:#313244; }"
    );
    searchBar->setStyleSheet(
        "QLineEdit { background:#1e1e2e; color:#cdd6f4; border:1px solid #313244; border-radius:6px; padding:6px; }"
    );
    emojiButton->setStyleSheet("QPushButton { background:transparent; border:none; font-size:18px; }");
    attachButton->setStyleSheet("QPushButton { background:transparent; border:none; font-size:18px; }");
}

void MainWindow::applyTheme()
{
    // Catppuccin Mocha dark palette
    QPalette pal = palette();
    pal.setColor(QPalette::Window, QColor("#1e1e2e"));
    pal.setColor(QPalette::WindowText, QColor("#cdd6f4"));
    pal.setColor(QPalette::Base, QColor("#11111b"));
    pal.setColor(QPalette::Text, QColor("#cdd6f4"));
    setPalette(pal);
}

void MainWindow::setupMenu()
{
    QMenuBar *menuBar = this->menuBar();
    QMenu *helpMenu = menuBar->addMenu("Help");
    QAction *aboutAct = helpMenu->addAction("About");
    connect(aboutAct, &QAction::triggered, this, [this](){
        QMessageBox::about(this, "About FynChat Desktop",
            "FynChat Desktop\nVersion 0.1.0\nQt Desktop client for FynChat\n\nBuilt with Qt 6");
    });
}

void MainWindow::showAuthDialog()
{
    if (!m_client) {
        return;
    }
    LoginDialog dialog(m_client, this);

    // Wait for server auth response before closing dialog
    QObject::connect(m_client, &FynClient::commandReceived, this, [&](CommandType cmd, const QByteArray &payload){
        if (cmd == CommandType::Login || cmd == CommandType::Register) {
            QString resp = QString::fromUtf8(payload).trimmed();
            if (resp.isEmpty() || resp.compare("OK", Qt::CaseInsensitive) == 0) {
                m_loggedIn = true;
                emit authSucceeded();
                dialog.accept();
            } else {
                QMessageBox::warning(this, "Authentication failed", resp.isEmpty() ? "Unknown error" : resp);
            }
        }
    });

    dialog.exec();
}
