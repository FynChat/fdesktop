#pragma once
#include <QMainWindow>
#include <QWidget>

class QVBoxLayout;
class QHBoxLayout;
class QListWidget;
class QTextEdit;
class QPushButton;
class QLabel;
class QLineEdit;
class QSplitter;

class FynClient;

class MainWindow : public QMainWindow {
    Q_OBJECT
public:
    explicit MainWindow(FynClient *client = nullptr, QWidget *parent = nullptr);
    void addChatMessage(const QString &text, bool incoming = true);
    QTextEdit* inputEdit() { return inputArea; }
    QLabel* headerLabel() { return chatHeader; }

signals:
    void authSucceeded();

private:
    void setupUI();
    void applyTheme();
    void styleWidgets();
    void setupMenu();
    void showAuthDialog();

    FynClient *m_client;
    bool m_loggedIn;

    QWidget *sidebar;
    QWidget *chatPane;
    QListWidget *chatList;
    QLineEdit *searchBar;
    QLabel *avatarLabel;
    QLabel *chatHeader;
    QListWidget *chatHistory;
    QTextEdit *inputArea;
    QPushButton *sendButton;
    QPushButton *emojiButton;
    QPushButton *attachButton;
    QSplitter *splitter;
};
