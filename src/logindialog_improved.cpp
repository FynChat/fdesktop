#include "logindialog_improved.hpp"
#include "fynclient.hpp"
#include "shared/protocol.hpp"
#include <QFrame>
#include <QHBoxLayout>
#include <QLabel>
#include <QLineEdit>
#include <QMessageBox>
#include <QPushButton>
#include <QVBoxLayout>

LoginDialogImproved::LoginDialogImproved(FynClient *client, QWidget *parent)
    : QDialog(parent), m_client(client) {
  setWindowTitle("FynChat Login");
  setModal(true);
  resize(400, 360);
  setupUI();
  applyTheme();
}

void LoginDialogImproved::setupUI() {
  QVBoxLayout *mainLayout = new QVBoxLayout(this);
  mainLayout->setContentsMargins(0, 0, 0, 0);
  mainLayout->setSpacing(0);

  // Card container
  QFrame *card = new QFrame();
  card->setObjectName("card");
  QVBoxLayout *layout = new QVBoxLayout(card);
  layout->setContentsMargins(32, 32, 32, 32);
  layout->setSpacing(16);

  QLabel *title = new QLabel("Welcome to FynChat");
  title->setObjectName("dialogTitle");
  title->setAlignment(Qt::AlignCenter);

  QLabel *subtitle = new QLabel("Sign in to continue");
  subtitle->setAlignment(Qt::AlignCenter);
  subtitle->setStyleSheet("color:#6c7086;");

  m_usernameEdit = new QLineEdit();
  m_usernameEdit->setPlaceholderText("Username");
  m_usernameEdit->setClearButtonEnabled(true);

  m_passwordEdit = new QLineEdit();
  m_passwordEdit->setPlaceholderText("Password");
  m_passwordEdit->setEchoMode(QLineEdit::Password);
  m_passwordEdit->setClearButtonEnabled(true);

  QHBoxLayout *buttonLayout = new QHBoxLayout();
  m_registerButton = new QPushButton("Register");
  m_loginButton = new QPushButton("Login");
  m_loginButton->setDefault(true);
  buttonLayout->addWidget(m_registerButton);
  buttonLayout->addWidget(m_loginButton);

  layout->addWidget(title);
  layout->addWidget(subtitle);
  layout->addSpacing(8);
  layout->addWidget(m_usernameEdit);
  layout->addWidget(m_passwordEdit);
  layout->addSpacing(12);
  layout->addLayout(buttonLayout);

  mainLayout->addWidget(card);

  connect(m_loginButton, &QPushButton::clicked, this,
          &LoginDialogImproved::onLoginClicked);
  connect(m_registerButton, &QPushButton::clicked, this,
          &LoginDialogImproved::onRegisterClicked);
}

void LoginDialogImproved::applyTheme() {
  setStyleSheet(
      "QDialog { background:#1e1e2e; }"
      "#card { background:#11111b; border:1px solid #313244; "
      "border-radius:16px; }"
      "#dialogTitle { font-size:20px; font-weight:700; margin-bottom:4px; "
      "color:#cdd6f4; }"
      "QLineEdit { background:#1e1e2e; color:#cdd6f4; border:1px solid "
      "#313244; border-radius:8px; padding:10px; font-size:14px; }"
      "QLineEdit:focus { border:1px solid #89b4fa; }"
      "QPushButton { background:#89b4fa; color:#1e1e2e; border:none; "
      "border-radius:8px; padding:10px 20px; font-weight:600; font-size:14px; }"
      "QPushButton:hover { background:#74c7ec; }"
      "QPushButton#registerBtn, QPushButton { }"
      "QPushButton[objectName=\"registerButton\"] { background:#fab387; "
      "color:#1e1e2e; }"
      "QPushButton[objectName=\"registerButton\"]:hover { background:#f5c2e7; "
      "}");
  m_registerButton->setObjectName("registerButton");
}

void LoginDialogImproved::onLoginClicked() {
  QString username = m_usernameEdit->text().trimmed();
  QString password = m_passwordEdit->text();

  if (username.isEmpty() || password.isEmpty()) {
    QMessageBox::warning(this, "Input required",
                         "Username and password are required.");
    return;
  }

  if (m_client) {
    m_client->sendCommand(CommandType::Login, username + ":" + password);
  } else {
    QMessageBox::warning(this, "Error", "No client available");
  }
  // Wait for server response before closing
}

void LoginDialogImproved::onRegisterClicked() {
  QString username = m_usernameEdit->text().trimmed();
  QString password = m_passwordEdit->text();

  if (username.isEmpty() || password.isEmpty()) {
    QMessageBox::warning(this, "Input required",
                         "Username is required to register.");
    return;
  }

  if (m_client) {
    m_client->sendCommand(CommandType::Register, username, password);
  } else {
    QMessageBox::warning(this, "Error", "No client available");
  }
  // Wait for server response before closing
}
