// Copyright2018-2020 Vikunja and contriubtors. All rights reserved.
//
// This file is part of Vikunja.
//
// Vikunja is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// Vikunja is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with Vikunja.  If not, see <https://www.gnu.org/licenses/>.

package user

import (
	"code.vikunja.io/api/pkg/config"
	"code.vikunja.io/api/pkg/mail"
	"code.vikunja.io/api/pkg/utils"
)

// PasswordReset holds the data to reset a password
type PasswordReset struct {
	// The previously issued reset token.
	Token string `json:"token"`
	// The new password for this user.
	NewPassword string `json:"new_password"`
}

// ResetPassword resets a users password
func ResetPassword(reset *PasswordReset) (err error) {

	// Check if the password is not empty
	if reset.NewPassword == "" {
		return ErrNoUsernamePassword{}
	}

	// Check if we have a token
	var user User
	exists, err := x.Where("password_reset_token = ?", reset.Token).Get(&user)
	if err != nil {
		return
	}

	if !exists {
		return ErrInvalidPasswordResetToken{Token: reset.Token}
	}

	// Hash the password
	user.Password, err = hashPassword(reset.NewPassword)
	if err != nil {
		return
	}

	// Save it
	_, err = x.Where("id = ?", user.ID).Update(&user)
	if err != nil {
		return
	}

	// Dont send a mail if we're testing
	if !config.MailerEnabled.GetBool() {
		return
	}

	// Send a mail to the user to notify it his password was changed.
	data := map[string]interface{}{
		"User": user,
	}

	mail.SendMailWithTemplate(user.Email, "Your password on Vikunja was changed", "password-changed", data)

	return
}

// PasswordTokenRequest defines the request format for password reset resqest
type PasswordTokenRequest struct {
	Email string `json:"email" valid:"email,length(0|250)" maxLength:"250"`
}

// RequestUserPasswordResetTokenByEmail inserts a random token to reset a users password into the databsse
func RequestUserPasswordResetTokenByEmail(tr *PasswordTokenRequest) (err error) {
	if tr.Email == "" {
		return ErrNoUsernamePassword{}
	}

	// Check if the user exists
	user, err := GetUserWithEmail(&User{Email: tr.Email})
	if err != nil {
		return
	}

	return RequestUserPasswordResetToken(user)
}

// RequestUserPasswordResetToken sends a user a password reset email.
func RequestUserPasswordResetToken(user *User) (err error) {
	// Generate a token and save it
	user.PasswordResetToken = utils.MakeRandomString(400)

	// Save it
	_, err = x.Where("id = ?", user.ID).Update(user)
	if err != nil {
		return
	}

	// Dont send a mail if we're testing
	if !config.MailerEnabled.GetBool() {
		return
	}

	data := map[string]interface{}{
		"User": user,
	}

	// Send the user a mail with the reset token
	mail.SendMailWithTemplate(user.Email, "Reset your password on Vikunja", "reset-password", data)
	return
}
