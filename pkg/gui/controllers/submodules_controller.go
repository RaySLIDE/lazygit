package controllers

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/jesseduffield/lazygit/pkg/commands"
	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/jesseduffield/lazygit/pkg/common"
	"github.com/jesseduffield/lazygit/pkg/config"
	"github.com/jesseduffield/lazygit/pkg/gui/filetree"
	"github.com/jesseduffield/lazygit/pkg/gui/popup"
	"github.com/jesseduffield/lazygit/pkg/gui/style"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
)

type SubmodulesController struct {
	*common.Common
	IGuiCommon
	enterSubmoduleFn     func(submodule *models.SubmoduleConfig) error
	getSelectedSubmodule func() *models.SubmoduleConfig
	git                  *commands.GitCommand
	fileManager          *filetree.FileManager
	submodules           []*models.SubmoduleConfig
}

func NewSubmodulesController(
	common *common.Common,
	guiCommon IGuiCommon,
	enterSubmoduleFn func(submodule *models.SubmoduleConfig) error,
	git *commands.GitCommand,
	fileManager *filetree.FileManager,
	submodules []*models.SubmoduleConfig,
	getSelectedSubmodule func() *models.SubmoduleConfig,
) *SubmodulesController {
	return &SubmodulesController{
		Common:               common,
		IGuiCommon:           guiCommon,
		enterSubmoduleFn:     enterSubmoduleFn,
		git:                  git,
		fileManager:          fileManager,
		submodules:           submodules,
		getSelectedSubmodule: getSelectedSubmodule,
	}
}

func (self *SubmodulesController) Keybindings(getKey func(key string) interface{}, config config.KeybindingConfig) []*types.Binding {
	return []*types.Binding{
		{
			Key:         getKey(config.Universal.GoInto),
			Handler:     self.forSubmodule(self.Enter),
			Description: self.Tr.LcEnterSubmodule,
		},
		{
			Key:         getKey(config.Universal.Remove),
			Handler:     self.forSubmodule(self.OpenResetMenu),
			Description: self.Tr.LcViewResetAndRemoveOptions,
			OpensMenu:   true,
		},
		{
			Key:         getKey(config.Submodules.Update),
			Handler:     self.forSubmodule(self.Update),
			Description: self.Tr.LcSubmoduleUpdate,
		},
		{
			Key:         getKey(config.Universal.New),
			Handler:     self.HandleAddSubmodule,
			Description: self.Tr.LcAddSubmodule,
		},
		{
			Key:         getKey(config.Universal.Edit),
			Handler:     self.forSubmodule(self.EditURL),
			Description: self.Tr.LcEditSubmoduleUrl,
		},
		{
			Key:         getKey(config.Submodules.Init),
			Handler:     self.forSubmodule(self.Init),
			Description: self.Tr.LcInitSubmodule,
		},
		{
			Key:         getKey(config.Submodules.BulkMenu),
			Handler:     self.OpenBulkActionsMenu,
			Description: self.Tr.LcViewBulkSubmoduleOptions,
			OpensMenu:   true,
		},
	}
}

func (self *SubmodulesController) Enter(submodule *models.SubmoduleConfig) error {
	return self.enterSubmoduleFn(submodule)
}

func (self *SubmodulesController) Reset(submodule *models.SubmoduleConfig) error {
	return self.WithWaitingStatus(self.Tr.LcResettingSubmoduleStatus, func() error {
		return self.resetSubmodule(submodule)
	})
}

func (self *SubmodulesController) HandleAddSubmodule() error {
	return self.Prompt(popup.PromptOpts{
		Title: self.Tr.LcNewSubmoduleUrl,
		HandleConfirm: func(submoduleUrl string) error {
			nameSuggestion := filepath.Base(strings.TrimSuffix(submoduleUrl, filepath.Ext(submoduleUrl)))

			return self.Prompt(popup.PromptOpts{
				Title:          self.Tr.LcNewSubmoduleName,
				InitialContent: nameSuggestion,
				HandleConfirm: func(submoduleName string) error {

					return self.Prompt(popup.PromptOpts{
						Title:          self.Tr.LcNewSubmodulePath,
						InitialContent: submoduleName,
						HandleConfirm: func(submodulePath string) error {
							return self.WithWaitingStatus(self.Tr.LcAddingSubmoduleStatus, func() error {
								self.LogAction(self.Tr.Actions.AddSubmodule)
								err := self.git.Submodule.Add(submoduleName, submodulePath, submoduleUrl)
								if err != nil {
									_ = self.Error(err)
								}

								return self.Refresh(types.RefreshOptions{Scope: []types.RefreshableView{types.SUBMODULES}})
							})
						},
					})
				},
			})
		},
	})
}

func (self *SubmodulesController) EditURL(submodule *models.SubmoduleConfig) error {
	return self.Prompt(popup.PromptOpts{
		Title:          fmt.Sprintf(self.Tr.LcUpdateSubmoduleUrl, submodule.Name),
		InitialContent: submodule.Url,
		HandleConfirm: func(newUrl string) error {
			return self.WithWaitingStatus(self.Tr.LcUpdatingSubmoduleUrlStatus, func() error {
				self.LogAction(self.Tr.Actions.UpdateSubmoduleUrl)
				err := self.git.Submodule.UpdateUrl(submodule.Name, submodule.Path, newUrl)
				if err != nil {
					_ = self.Error(err)
				}

				return self.Refresh(types.RefreshOptions{Scope: []types.RefreshableView{types.SUBMODULES}})
			})
		},
	})
}

func (self *SubmodulesController) Init(submodule *models.SubmoduleConfig) error {
	return self.WithWaitingStatus(self.Tr.LcInitializingSubmoduleStatus, func() error {
		self.LogAction(self.Tr.Actions.InitialiseSubmodule)
		err := self.git.Submodule.Init(submodule.Path)
		if err != nil {
			_ = self.Error(err)
		}

		return self.Refresh(types.RefreshOptions{Scope: []types.RefreshableView{types.SUBMODULES}})
	})
}

func (self *SubmodulesController) OpenResetMenu(submodule *models.SubmoduleConfig) error {
	return self.Menu(popup.CreateMenuOptions{
		Title: submodule.Name,
		Items: []*popup.MenuItem{
			{
				DisplayString: self.Tr.LcSubmoduleStashAndReset,
				OnPress: func() error {
					return self.resetSubmodule(submodule)
				},
			},
			{
				DisplayString: self.Tr.LcRemoveSubmodule,
				OnPress: func() error {
					return self.removeSubmodule(submodule)
				},
			},
		},
	})
}

func (self *SubmodulesController) OpenBulkActionsMenu() error {
	return self.Menu(popup.CreateMenuOptions{
		Title: self.Tr.LcBulkSubmoduleOptions,
		Items: []*popup.MenuItem{
			{
				DisplayStrings: []string{self.Tr.LcBulkInitSubmodules, style.FgGreen.Sprint(self.git.Submodule.BulkInitCmdObj().ToString())},
				OnPress: func() error {
					return self.WithWaitingStatus(self.Tr.LcRunningCommand, func() error {
						self.LogAction(self.Tr.Actions.BulkInitialiseSubmodules)
						err := self.git.Submodule.BulkInitCmdObj().Run()
						if err != nil {
							return self.Error(err)
						}

						return self.Refresh(types.RefreshOptions{Scope: []types.RefreshableView{types.SUBMODULES}})
					})
				},
			},
			{
				DisplayStrings: []string{self.Tr.LcBulkUpdateSubmodules, style.FgYellow.Sprint(self.git.Submodule.BulkUpdateCmdObj().ToString())},
				OnPress: func() error {
					return self.WithWaitingStatus(self.Tr.LcRunningCommand, func() error {
						self.LogAction(self.Tr.Actions.BulkUpdateSubmodules)
						if err := self.git.Submodule.BulkUpdateCmdObj().Run(); err != nil {
							return self.Error(err)
						}

						return self.Refresh(types.RefreshOptions{Scope: []types.RefreshableView{types.SUBMODULES}})
					})
				},
			},
			{
				DisplayStrings: []string{self.Tr.LcSubmoduleStashAndReset, style.FgRed.Sprintf("git stash in each submodule && %s", self.git.Submodule.ForceBulkUpdateCmdObj().ToString())},
				OnPress: func() error {
					return self.WithWaitingStatus(self.Tr.LcRunningCommand, func() error {
						self.LogAction(self.Tr.Actions.BulkStashAndResetSubmodules)
						if err := self.git.Submodule.ResetSubmodules(self.submodules); err != nil {
							return self.Error(err)
						}

						return self.Refresh(types.RefreshOptions{Scope: []types.RefreshableView{types.SUBMODULES}})
					})
				},
			},
			{
				DisplayStrings: []string{self.Tr.LcBulkDeinitSubmodules, style.FgRed.Sprint(self.git.Submodule.BulkDeinitCmdObj().ToString())},
				OnPress: func() error {
					return self.WithWaitingStatus(self.Tr.LcRunningCommand, func() error {
						self.LogAction(self.Tr.Actions.BulkDeinitialiseSubmodules)
						if err := self.git.Submodule.BulkDeinitCmdObj().Run(); err != nil {
							return self.Error(err)
						}

						return self.Refresh(types.RefreshOptions{Scope: []types.RefreshableView{types.SUBMODULES}})
					})
				},
			},
		},
	})
}

func (self *SubmodulesController) Update(submodule *models.SubmoduleConfig) error {
	return self.WithWaitingStatus(self.Tr.LcUpdatingSubmoduleStatus, func() error {
		self.LogAction(self.Tr.Actions.UpdateSubmodule)
		err := self.git.Submodule.Update(submodule.Path)
		if err != nil {
			_ = self.Error(err)
		}

		return self.Refresh(types.RefreshOptions{Scope: []types.RefreshableView{types.SUBMODULES}})
	})
}

func (self *SubmodulesController) FileForSubmodule(submodule *models.SubmoduleConfig) *models.File {
	for _, file := range self.fileManager.GetAllFiles() {
		if file.IsSubmodule([]*models.SubmoduleConfig{submodule}) {
			return file
		}
	}

	return nil
}

func (self *SubmodulesController) removeSubmodule(submodule *models.SubmoduleConfig) error {
	return self.Ask(popup.AskOpts{
		Title:  self.Tr.RemoveSubmodule,
		Prompt: fmt.Sprintf(self.Tr.RemoveSubmodulePrompt, submodule.Name),
		HandleConfirm: func() error {
			self.LogAction(self.Tr.Actions.RemoveSubmodule)
			if err := self.git.Submodule.Delete(submodule); err != nil {
				return self.Error(err)
			}

			return self.Refresh(types.RefreshOptions{Scope: []types.RefreshableView{types.SUBMODULES, types.FILES}})
		},
	})
}

func (self *SubmodulesController) resetSubmodule(submodule *models.SubmoduleConfig) error {
	self.LogAction(self.Tr.Actions.ResetSubmodule)

	file := self.FileForSubmodule(submodule)
	if file != nil {
		if err := self.git.WorkingTree.UnStageFile(file.Names(), file.Tracked); err != nil {
			return self.Error(err)
		}
	}

	if err := self.git.Submodule.Stash(submodule); err != nil {
		return self.Error(err)
	}
	if err := self.git.Submodule.Reset(submodule); err != nil {
		return self.Error(err)
	}

	return self.Refresh(types.RefreshOptions{Mode: types.ASYNC, Scope: []types.RefreshableView{types.FILES, types.SUBMODULES}})
}

func (self *SubmodulesController) forSubmodule(callback func(*models.SubmoduleConfig) error) func() error {
	return func() error {
		submodule := self.getSelectedSubmodule()
		if submodule == nil {
			return nil
		}

		return callback(submodule)
	}
}
