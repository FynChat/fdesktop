#pragma once
#include <QDialog>

class QLineEdit;
class QPushButton;
class QLabel;
class QVBoxLayout;
class QHBoxLayout;

class FynClient;

class LoginDialog : public QDialog {
    Q_OBJECT
public:
    explicit LoginDialog(FynClient* client, QWidget *parent = nullptr);

signals:
    void authSucceeded();

private slots:
    void onLoginClicked();
    void onRegisterClicked();

private:
    void setupUI();
    void applyTheme();

    FynClient *m_client;
    QLineEdit *m_usernameEdit;
    QLineEdit *m_passwordEdit;
    QPushButton *m_loginButton;
    QPushButton *m_registerButton;
};
