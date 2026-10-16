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
#include <QPixmap>
#include <QToolButton>
#include <QTextBrowser>
#include <QTimer>

MainWindow::MainWindow(FynClient *client, QWidget *parent)
    : QMainWindow(parent), m_client(client), m_loggedIn(false), sidebarExpanded(true)
{
    setWindowTitle("FynChat Desktop v0.2.0");
    resize(1100, 750);
    setupUI();
    applyTheme();
    setupMenu();

    if (!m_loggedIn) {
        showAuthDialog();
    }
}

void MainWindow::addChatMessage(const QString &text, bool incoming)
{
    QString bubbleColor = incoming ? "#313244" : "#cba6f7";
    QString align = incoming ? "left" : "right";
    QString html = QString(R"(
        <div style="display:flex; justify-content:%1; margin:6px 12px;">
            <div style="max-width:70%%; background:%2; color:#cdd6f4; padding:10px 14px; border-radius:12px; font-size:14px; line-height:1.4;">
                %3
            </div>
        </div>
    )").arg(align, bubbleColor, text.toHtmlEscaped());
    chatHistory->append(html);
    QTimer::singleShot(0, this, [this](){ chatHistory->scrollToAnchor("bottom"); });
}

void MainWindow::setupUI()
{
    QWidget *central = new QWidget(this);
    QHBoxLayout *mainLayout = new QHBoxLayout(central);
    mainLayout->setContentsMargins(0,0,0,0);
    mainLayout->setSpacing(0);

    splitter = new QSplitter(Qt::Horizontal, this);

    // Icon Tray
    iconTray = new QWidget();
    iconTray->setFixedWidth(56);
    QVBoxLayout *trayLayout = new QVBoxLayout(iconTray);
    trayLayout->setContentsMargins(8,12,8,12);
    trayLayout->setSpacing(12);
    toggleSidebarBtn = new QPushButton("☰");
    toggleSidebarBtn->setFixedSize(40,40);
    connect(toggleSidebarBtn, &QPushButton::clicked, this, &MainWindow::toggleSidebar);
    trayLayout->addWidget(toggleSidebarBtn, 0, Qt::AlignCenter);
    trayLayout->addSpacing(12);
    QPushButton *profileBtn = new QPushButton("👤");
    profileBtn->setFixedSize(40,40);
    trayLayout->addWidget(profileBtn, 0, Qt::AlignCenter);
    trayLayout->addStretch();

    // Sidebar
    sidebar = new QWidget();
    sidebar->setMinimumWidth(260);
    sidebar->setMaximumWidth(320);
    QVBoxLayout *sidebarLayout = new QVBoxLayout(sidebar);
    sidebarLayout->setContentsMargins(12,12,12,12);
    sidebarLayout->setSpacing(8);

    QLabel *sidebarTitle = new QLabel("Chats");
    sidebarTitle->setStyleSheet("font-weight:600; font-size:16px; color:#cdd6f4;");
    searchBar = new QLineEdit();
    searchBar->setPlaceholderText("Search chats...");
    searchBar->setClearButtonEnabled(true);

    chatList = new QListWidget();
    chatList->addItem("General");
    chatList->addItem("Direct Messages");
    chatList->addItem("Support");

    sidebarLayout->addWidget(sidebarTitle);
    sidebarLayout->addWidget(searchBar);
    sidebarLayout->addWidget(chatList);
    sidebarLayout->addStretch();

    // Chat Pane
    chatPane = new QWidget();
    QVBoxLayout *chatLayout = new QVBoxLayout(chatPane);
    chatLayout->setContentsMargins(0,0,0,0);
    chatLayout->setSpacing(0);

    // Header
    QWidget *headerWidget = new QWidget();
    headerWidget->setStyleSheet("background:#1e1e2e; padding:12px; border-bottom:1px solid #313244;");
    QHBoxLayout *headerLayout = new QHBoxLayout(headerWidget);
    avatarLabel = new QLabel();
    avatarLabel->setFixedSize(36,36);
    avatarLabel->setStyleSheet("background:#313244; border-radius:18px;");
    avatarLabel->setPixmap(QPixmap(":/resources/avatar.png").scaled(36,36,Qt::KeepAspectRatio,Qt::SmoothTransformation));
    chatHeader = new QLabel("FynChat");
    chatHeader->setStyleSheet("font-weight:600; font-size:16px; color:#cdd6f4;");
    QLabel *statusLabel = new QLabel("● Online");
    statusLabel->setStyleSheet("color:#a6e3a1;");
    headerLayout->addWidget(avatarLabel);
    headerLayout->addWidget(chatHeader);
    headerLayout->addStretch();
    headerLayout->addWidget(statusLabel);

    // History
    chatHistory = new QTextBrowser();
    chatHistory->setOpenExternalLinks(false);
    chatHistory->setStyleSheet("QTextBrowser { background:#11111b; color:#cdd6f4; border:none; padding:8px; }");
    chatHistory->setHtml(R"(<body style="background:#11111b; font-family:sans-serif;">)");

    // Input
    QWidget *inputWidget = new QWidget();
    inputWidget->setStyleSheet("background:#1e1e2e; padding:8px; border-top:1px solid #313244;");
    QHBoxLayout *inputLayout = new QHBoxLayout(inputWidget);
    inputLayout->setSpacing(6);

    emojiButton = new QToolButton();
    emojiButton->setText("😊");
    emojiButton->setFixedSize(36,36);
    attachButton = new QToolButton();
    attachButton->setText("📎");
    attachButton->setFixedSize(36,36);
    connect(attachButton, &QToolButton::clicked, this, [this](){
        QString fileName = QFileDialog::getOpenFileName(this, "Select Avatar", "", "Images (*.png *.jpg *.jpeg)");
        if (!fileName.isEmpty() && m_client) {
            QFile file(fileName);
            const qint64 MAX_SIZE = 5 * 1024 * 1024;
            if (!file.exists() || file.size() > MAX_SIZE) {
                QMessageBox::warning(this, "File too large", "Avatar must be smaller than 5 MB");
                return;
            }
            if (file.open(QIODevice::ReadOnly)) {
                QByteArray data = file.readAll();
                QByteArray base64 = data.toBase64();
                m_client->sendCommand(CommandType::AvatarUpload, base64);
                QPixmap pm(fileName);
                if (!pm.isNull()) avatarLabel->setPixmap(pm.scaled(36,36,Qt::KeepAspectRatio,Qt::SmoothTransformation));
            }
        }
    });

    inputArea = new QTextEdit();
    inputArea->setPlaceholderText("Type a message...");
    inputArea->setMaximumHeight(120);
    inputArea->setStyleSheet("background:#11111b; color:#cdd6f4; border:1px solid #313244; border-radius:8px; padding:8px;");

    sendButton = new QPushButton("Send");
    sendButton->setObjectName("sendButton");
    sendButton->setMinimumWidth(80);
    sendButton->setStyleSheet(
        "QPushButton { background:#89b4fa; color:#1e1e2e; border:none; border-radius:8px; padding:8px 16px; font-weight:600; }"
        "QPushButton:hover { background:#74c7ec; }"
    );

    inputLayout->addWidget(emojiButton);
    inputLayout->addWidget(attachButton);
    inputLayout->addWidget(inputArea, 1);
    inputLayout->addWidget(sendButton);

    chatLayout->addWidget(headerWidget);
    chatLayout->addWidget(chatHistory, 1);
    chatLayout->addWidget(inputWidget);

    splitter->addWidget(iconTray);
    splitter->addWidget(sidebar);
    splitter->addWidget(chatPane);
    splitter->setSizes({56, 260, 784});
    splitter->setCollapsible(0, false);
    splitter->setCollapsible(1, true);

    mainLayout->addWidget(splitter);
    setCentralWidget(central);

    styleWidgets();
}

void MainWindow::toggleSidebar()
{
    sidebarExpanded = !sidebarExpanded;
    sidebar->setVisible(sidebarExpanded);
    splitter->setSizes({56, sidebarExpanded ? 260 : 0, 784});
}

void MainWindow::styleWidgets()
{
    sidebar->setStyleSheet("background:#11111b; color:#cdd6f4;");
    chatList->setStyleSheet(
        "QListWidget { background:#11111b; border:none; }"
        "QListWidget::item { padding:10px; border-radius:8px; }"
        "QListWidget::item:selected { background:#313244; }"
    );
    searchBar->setStyleSheet(
        "QLineEdit { background:#1e1e2e; color:#cdd6f4; border:1px solid #313244; border-radius:8px; padding:8px; }"
    );
    iconTray->setStyleSheet("background:#11111b;");
    toggleSidebarBtn->setStyleSheet("QPushButton { background:#1e1e2e; border:none; border-radius:20px; font-size:18px; } QPushButton:hover { background:#313244; }");
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
            "FynChat Desktop\nVersion 0.2.0\nQt Desktop client for FynChat\n\nBuilt with Qt 6");
    });
}

void MainWindow::showAuthDialog()
{
    if (!m_client) return;
    LoginDialog dialog(m_client, this);

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
