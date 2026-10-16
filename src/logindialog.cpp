#include "logindialog.hpp"
#include "fynclient.hpp"
#include "shared/protocol.hpp"
#include <QVBoxLayout>
#include <QHBoxLayout>
#include <QLabel>
#include <QLineEdit>
#include <QPushButton>
#include <QMessageBox>

LoginDialog::LoginDialog(FynClient *client, QWidget *parent)
    : QDialog(parent), m_client(client)
{
    setWindowTitle("FynChat Login");
    setModal(true);
    resize(360, 280);
    setupUI();
    applyTheme();
}

void LoginDialog::setupUI()
{
    QVBoxLayout *layout = new QVBoxLayout(this);
    layout->setContentsMargins(24, 24, 24, 24);
    layout->setSpacing(12);

    QLabel *title = new QLabel("Welcome to FynChat");
    title->setObjectName("dialogTitle");
    title->setAlignment(Qt::AlignCenter);

    m_usernameEdit = new QLineEdit();
    m_usernameEdit->setPlaceholderText("Username");

    m_passwordEdit = new QLineEdit();
    m_passwordEdit->setPlaceholderText("Password");
    m_passwordEdit->setEchoMode(QLineEdit::Password);

    QHBoxLayout *buttonLayout = new QHBoxLayout();
    m_registerButton = new QPushButton("Register");
    m_loginButton = new QPushButton("Login");
    buttonLayout->addWidget(m_registerButton);
    buttonLayout->addWidget(m_loginButton);

    layout->addWidget(title);
    layout->addWidget(m_usernameEdit);
    layout->addWidget(m_passwordEdit);
    layout->addStretch();
    layout->addLayout(buttonLayout);

    connect(m_loginButton, &QPushButton::clicked, this, &LoginDialog::onLoginClicked);
    connect(m_registerButton, &QPushButton::clicked, this, &LoginDialog::onRegisterClicked);
}

void LoginDialog::applyTheme()
{
    m_registerButton->setObjectName("registerButton");
    setStyleSheet(
        "QDialog { background:#1e1e2e; color:#cdd6f4; }"
        "#dialogTitle { font-size:18px; font-weight:600; margin-bottom:8px; }"
        "QLineEdit { background:#11111b; color:#cdd6f4; border:1px solid #313244; border-radius:6px; padding:8px; }"
        "QPushButton { background:#89b4fa; color:#1e1e2e; border:none; border-radius:6px; padding:8px 16px; font-weight:600; }"
        "QPushButton:hover { background:#74c7ec; }"
        "QPushButton#registerButton { background:#fab387; }"
    );
}

void LoginDialog::onLoginClicked()
{
    QString username = m_usernameEdit->text().trimmed();
    QString password = m_passwordEdit->text();

    if (username.isEmpty() || password.isEmpty()) {
        QMessageBox::warning(this, "Input required", "Username and password are required.");
        return;
    }

    if (m_client) {
        m_client->sendCommand(CommandType::Login, username + ":" + password);
    } else {
        QMessageBox::warning(this, "Error", "No client available");
    }
    // Do not accept here – wait for server response
}

void LoginDialog::onRegisterClicked()
{
    QString username = m_usernameEdit->text().trimmed();

    if (username.isEmpty()) {
        QMessageBox::warning(this, "Input required", "Username is required to register.");
        return;
    }

    if (m_client) {
        m_client->sendCommand(CommandType::Register, username);
    } else {
        QMessageBox::warning(this, "Error", "No client available");
    }
    // Do not accept here – wait for server response
}
