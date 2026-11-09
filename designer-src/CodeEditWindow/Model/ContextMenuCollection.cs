using System;
using System.Collections.Generic;
using System.Linq;
using System.Windows;
using System.Windows.Input;
using SpecDesigner.CodeEditWindow.Helper;
using SpecDesigner.Infrastructure;
using SpecDesigner.Infrastructure.Model;

namespace SpecDesigner.CodeEditWindow.Model
{
	// Token: 0x0200002C RID: 44
	public class ContextMenuCollection : IDisposable
	{
		// Token: 0x17000039 RID: 57
		// (get) Token: 0x060001AE RID: 430 RVA: 0x0000EDA4 File Offset: 0x0000CFA4
		public List<CommandModel> Commands
		{
			get
			{
				return this._commands;
			}
		}

		// Token: 0x060001AF RID: 431 RVA: 0x0000EDAC File Offset: 0x0000CFAC
		public ContextMenuCollection()
		{
			this.LoadCommands();
			CommandModel commandModel = new CommandModel(CodeEditCommands.FramMark, Application.Current.FindResource("CE_MarkFrame") as string);
			commandModel.IsCheckable = true;
			this._commands.Add(commandModel);
			CommandModel commandModel2 = new CommandModel(CodeEditCommands.CiteOrNot, Application.Current.FindResource("CE_CiteOrNot") as string);
			commandModel2.IsCheckable = true;
			this._commands.Add(commandModel2);
			this._commands.Add(null);
			CommandModel commandModel3 = new CommandModel(CodeEditCommands.ShowAddPointName, string.Format("AddPoint Name: {0}", "--------"));
			this._commands.Add(commandModel3);
		}

		// Token: 0x060001B0 RID: 432 RVA: 0x0000EEA4 File Offset: 0x0000D0A4
		public void SetAddPoint(AddPointModel model)
		{
			this._model = model;
			CommandModel commandModel = this._commands.Where<CommandModel>((CommandModel cmd) => cmd != null && cmd.Command == CodeEditCommands.FramMark).FirstOrDefault<CommandModel>();
			if (commandModel != null && model != null && model.IsMarkHard)
			{
				commandModel.IsChecked = true;
			}
			CommandModel commandModel2 = this._commands.Where<CommandModel>((CommandModel cmd) => cmd != null && cmd.Command == CodeEditCommands.CiteOrNot).FirstOrDefault<CommandModel>();
			if (commandModel2 != null && model != null && model.CiteSetting == YesNo.Y)
			{
				commandModel2.IsChecked = true;
			}
			CommandModel commandModel3 = this._commands.Where<CommandModel>((CommandModel cmd) => cmd != null && cmd.Command == CodeEditCommands.ShowAddPointName).FirstOrDefault<CommandModel>();
			if (commandModel3 != null)
			{
				commandModel3.Text = ((model != null) ? model.Name : "--------");
			}
		}

		// Token: 0x060001B1 RID: 433 RVA: 0x0000EF88 File Offset: 0x0000D188
		private ContextMenuCollection(AddPointModel model)
		{
			this._model = model;
			this.LoadCommands();
			CommandModel commandModel = new CommandModel(CodeEditCommands.FramMark, Application.Current.FindResource("CE_MarkFrame") as string);
			commandModel.IsCheckable = true;
			if (model != null && model.IsMarkHard)
			{
				commandModel.IsChecked = true;
			}
			this._commands.Add(commandModel);
			if (model != null && !ResourceController.GetInstance().GetProgramInfo(model.ProgramKey).IsStandard)
			{
				CommandModel commandModel2 = new CommandModel(CodeEditCommands.CiteOrNot, Application.Current.FindResource("CE_CiteOrNot") as string);
				commandModel2.IsCheckable = true;
				if (model != null && model.CiteSetting == YesNo.Y)
				{
					commandModel2.IsChecked = true;
				}
				this._commands.Add(commandModel2);
			}
			this._commands.Add(null);
			string text = ((model != null) ? model.Name : "--------");
			CommandModel commandModel3 = new CommandModel(CodeEditCommands.ShowAddPointName, string.Format("AddPoint Name: {0}", text));
			this._commands.Add(commandModel3);
		}

		// Token: 0x060001B2 RID: 434 RVA: 0x0000F098 File Offset: 0x0000D298
		public void CreateContextMenu()
		{
			CommandModel commandModel = new CommandModel(ApplicationCommands.Cut, Application.Current.FindResource("CE_Cut") as string);
			this._commands.Add(commandModel);
		}

		// Token: 0x060001B3 RID: 435 RVA: 0x0000F0D0 File Offset: 0x0000D2D0
		private void LoadCommands()
		{
			CommandModel commandModel = new CommandModel(ApplicationCommands.Cut, "剪下");
			this._commands.Add(commandModel);
			CommandModel commandModel2 = new CommandModel(ApplicationCommands.Copy, "測試");
			this._commands.Add(commandModel2);
			CommandModel commandModel3 = new CommandModel(ApplicationCommands.Paste, "測試");
			this._commands.Add(commandModel3);
			this._commands.Add(null);
			CommandModel commandModel4 = new CommandModel(CodeEditCommands.Rename, "測試");
			this._commands.Add(commandModel4);
			CodeEditCommands.DeleteFunction.Parameter = this._model;
			CommandModel commandModel5 = new CommandModel(CodeEditCommands.DeleteFunction, "測試");
			this._commands.Add(commandModel5);
			CommandModel commandModel6 = new CommandModel(CodeEditCommands.BlockComment, "測試");
			this._commands.Add(commandModel6);
			CommandModel commandModel7 = new CommandModel(CodeEditCommands.UnblockComment, "測試");
			this._commands.Add(commandModel7);
			CommandModel commandModel8 = new CommandModel(CodeEditCommands.RefreshScreen, "測試");
			this._commands.Add(commandModel8);
			this._commands.Add(null);
			CommandModel commandModel9 = new CommandModel(CodeEditCommands.KeywordUpperCase, "測試");
			this._commands.Add(commandModel9);
			CommandModel commandModel10 = new CommandModel(CodeEditCommands.CodeCompletion, "測試");
			this._commands.Add(commandModel10);
			CommandModel commandModel11 = new CommandModel(CodeEditCommands.ShowFunction, "測試");
			this._commands.Add(commandModel11);
			CommandModel commandModel12 = new CommandModel(CodeEditCommands.InsertCodeSample, Application.Current.FindResource("CE_OftenUsed") as string);
			this._commands.Add(commandModel12);
		}

		// Token: 0x060001B4 RID: 436 RVA: 0x0000F274 File Offset: 0x0000D474
		public void Dispose()
		{
			this._commands.Clear();
		}

		// Token: 0x040000BA RID: 186
		private List<CommandModel> _commands = new List<CommandModel>();

		// Token: 0x040000BB RID: 187
		private AddPointModel _model;
	}
}
