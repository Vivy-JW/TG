using System;
using System.CodeDom.Compiler;
using System.Collections.Generic;
using System.ComponentModel;
using System.Diagnostics;
using System.Linq;
using System.Windows;
using System.Windows.Controls;
using System.Windows.Input;
using System.Windows.Markup;
using ICSharpCode.AvalonEdit;
using ICSharpCode.AvalonEdit.Document;
using ICSharpCode.AvalonEdit.Editing;
using SpecDesigner.CodeEditWindow.Helper;
using SpecDesigner.Infrastructure;
using SpecDesigner.Infrastructure.Helper;
using SpecDesigner.Infrastructure.Model;
using SpecDesignerCommon;
using SpecDesignerCommon.Events;

namespace SpecDesigner.CodeEditWindow.View
{
	// Token: 0x0200002B RID: 43
	public partial class DiffBaseOnStandardWindow : Window, IComparisonWindow
	{
		// Token: 0x17000032 RID: 50
		// (get) Token: 0x0600018E RID: 398 RVA: 0x0000E09A File Offset: 0x0000C29A
		public static DiffBaseOnStandardWindow This
		{
			get
			{
				if (DiffBaseOnStandardWindow._this == null)
				{
					DiffBaseOnStandardWindow._this = new DiffBaseOnStandardWindow();
				}
				return DiffBaseOnStandardWindow._this;
			}
		}

		// Token: 0x17000033 RID: 51
		// (get) Token: 0x0600018F RID: 399 RVA: 0x0000E0B2 File Offset: 0x0000C2B2
		// (set) Token: 0x06000190 RID: 400 RVA: 0x0000E0BA File Offset: 0x0000C2BA
		public PackageKey ProgramKey { get; private set; }

		// Token: 0x17000034 RID: 52
		// (get) Token: 0x06000191 RID: 401 RVA: 0x0000E0C3 File Offset: 0x0000C2C3
		// (set) Token: 0x06000192 RID: 402 RVA: 0x0000E0CB File Offset: 0x0000C2CB
		public AddPointModel currentAPT { get; private set; }

		// Token: 0x17000035 RID: 53
		// (get) Token: 0x06000193 RID: 403 RVA: 0x0000E0D4 File Offset: 0x0000C2D4
		// (set) Token: 0x06000194 RID: 404 RVA: 0x0000E0DC File Offset: 0x0000C2DC
		public DiffTextViewer textEditor1 { get; private set; }

		// Token: 0x17000036 RID: 54
		// (get) Token: 0x06000195 RID: 405 RVA: 0x0000E0E5 File Offset: 0x0000C2E5
		// (set) Token: 0x06000196 RID: 406 RVA: 0x0000E0ED File Offset: 0x0000C2ED
		public DiffTextViewer textEditor2 { get; private set; }

		// Token: 0x17000037 RID: 55
		// (get) Token: 0x06000197 RID: 407 RVA: 0x0000E0F6 File Offset: 0x0000C2F6
		// (set) Token: 0x06000198 RID: 408 RVA: 0x0000E0FE File Offset: 0x0000C2FE
		public DiffTextViewer SourceViewer
		{
			get
			{
				return this.textEditor1;
			}
			set
			{
				this.textEditor1 = value;
			}
		}

		// Token: 0x17000038 RID: 56
		// (get) Token: 0x06000199 RID: 409 RVA: 0x0000E107 File Offset: 0x0000C307
		// (set) Token: 0x0600019A RID: 410 RVA: 0x0000E10F File Offset: 0x0000C30F
		public DiffTextViewer TargetViewer
		{
			get
			{
				return this.textEditor2;
			}
			set
			{
				this.textEditor2 = value;
			}
		}

		// Token: 0x0600019B RID: 411 RVA: 0x0000E118 File Offset: 0x0000C318
		public DiffBaseOnStandardWindow()
		{
			this.InitializeComponent();
			base.CommandBindings.Add(new CommandBinding(DiffCommands.ToNextDiffLineCommand, new ExecutedRoutedEventHandler(this.OnToNextDiffLine), new CanExecuteRoutedEventHandler(this.CanToNextDiffLine)));
			base.CommandBindings.Add(new CommandBinding(DiffCommands.ToPreviousDiffLineCommand, new ExecutedRoutedEventHandler(this.OnToPreviousDiffLine), new CanExecuteRoutedEventHandler(this.CanToPreviousDiffLine)));
			base.CommandBindings.Add(new CommandBinding(DiffCommands.BaseOnStandardSaveCommand, new ExecutedRoutedEventHandler(this.OnBaseOnStandardSave), new CanExecuteRoutedEventHandler(this.CanBaseOnStandardSave)));
			base.CommandBindings.Add(new CommandBinding(CodeEditCommands.RefreshDiffResult, new ExecutedRoutedEventHandler(this.OnRefreshDiffResult)));
			EventAggregatorManager.Global.GetEvent<DiffBlockCopyContentEvent>().Subscribe(new Action<DiffCopyInfo>(this.OnDiffBlockCopy));
			EventAggregatorManager.Global.GetEvent<DiffLineCopyContentEvent>().Subscribe(new Action<DiffCopyInfo>(this.OnDiffLineCopy));
			Application.Current.Exit += this.Current_Exit;
		}

		// Token: 0x0600019C RID: 412 RVA: 0x0000E22B File Offset: 0x0000C42B
		private new void Show()
		{
			base.Show();
			if (base.Visibility != Visibility.Visible)
			{
				base.Visibility = Visibility.Visible;
			}
			base.Focus();
		}

		// Token: 0x0600019D RID: 413 RVA: 0x0000E24C File Offset: 0x0000C44C
		public void Show(string source, string target, PackageKey key, AddPointModel model)
		{
			this.ProgramKey = key;
			base.FindResource("VerticalGridSplitterStyle");
			base.FindResource("HorizontalGridSplitterStyle");
			Label label = new Label();
			Label label2 = new Label();
			this.textEditor1 = new DiffTextViewer(this.ProgramKey);
			this.textEditor2 = new DiffTextViewer(this.ProgramKey);
			label.SetValue(Grid.RowProperty, 0);
			this.textEditor1.SetValue(Grid.RowProperty, 1);
			this.viewer1.Children.Add(label);
			this.viewer1.Children.Add(this.textEditor1);
			label2.SetValue(Grid.RowProperty, 0);
			this.textEditor2.SetValue(Grid.RowProperty, 1);
			this.viewer2.Children.Add(label2);
			this.viewer2.Children.Add(this.textEditor2);
			base.Title = Application.Current.FindResource("CE_DiffBaseOnStandard") as string;
			label.Content = Application.Current.FindResource("CE_DiffSourceCustom") as string;
			label2.Content = Application.Current.FindResource("CE_DiffTargetStandars") as string;
			this.textEditor1.Text = source;
			this.textEditor2.Text = target;
			this.textEditor1.IsDiffBaseOnStandard = true;
			this.textEditor2.IsDiffBaseOnStandard = true;
			this.currentAPT = model;
			this._diffManager = new DiffManager(this.textEditor1, source, this.textEditor2, target, true, key);
			this.textEditor2.Mode = ContentType.OPEN;
			this.textEditor2.Document.SectionProvider.Attached();
		}

		// Token: 0x0600019E RID: 414 RVA: 0x0000E408 File Offset: 0x0000C608
		private void Current_Exit(object sender, ExitEventArgs e)
		{
			Application.Current.Exit -= this.Current_Exit;
			base.Close();
		}

		// Token: 0x0600019F RID: 415 RVA: 0x0000E428 File Offset: 0x0000C628
		protected override void OnClosing(CancelEventArgs e)
		{
			e.Cancel = true;
			base.Visibility = Visibility.Hidden;
			this.textEditor1.Dispose();
			this.textEditor2.Dispose();
			EventAggregatorManager.Global.GetEvent<DiffBlockCopyContentEvent>().Unsubscribe(new Action<DiffCopyInfo>(this.OnDiffBlockCopy));
			EventAggregatorManager.Global.GetEvent<DiffLineCopyContentEvent>().Unsubscribe(new Action<DiffCopyInfo>(this.OnDiffLineCopy));
		}

		// Token: 0x060001A0 RID: 416 RVA: 0x0000E4AC File Offset: 0x0000C6AC
		private void OnToNextDiffLine(object sender, ExecutedRoutedEventArgs e)
		{
			if (ResourceController.GetInstance().GetProgramInfo(this.ProgramKey).BaseOnStandardDiffColorArea != null)
			{
				DocumentLine CurrentLine = this.textEditor2.Document.GetLineByOffset(this.textEditor2.CaretOffset);
				DocumentLine documentLine = null;
				List<DiffColor> baseOnStandardDiffColorArea = ResourceController.GetInstance().GetProgramInfo(this.ProgramKey).BaseOnStandardDiffColorArea;
				if (baseOnStandardDiffColorArea.Count<DiffColor>() > 0)
				{
					if (ResourceController.GetInstance().GetProgramInfo(this.ProgramKey).BaseOnStandardDiffColorArea.Where<DiffColor>((DiffColor a) => a.LineNumber == CurrentLine.LineNumber).FirstOrDefault<DiffColor>() == null)
					{
						for (int i = 0; i < baseOnStandardDiffColorArea.Count<DiffColor>(); i++)
						{
							if (baseOnStandardDiffColorArea[i].LineNumber > CurrentLine.LineNumber)
							{
								documentLine = this.textEditor2.Document.GetLineByNumber(baseOnStandardDiffColorArea[i].LineNumber);
								break;
							}
						}
					}
					else
					{
						for (int j = 0; j < baseOnStandardDiffColorArea.Count<DiffColor>(); j++)
						{
							if (baseOnStandardDiffColorArea[j].LineNumber == CurrentLine.LineNumber)
							{
								for (int k = j + 1; k < baseOnStandardDiffColorArea.Count<DiffColor>(); k++)
								{
									if (baseOnStandardDiffColorArea[k].LineNumber != baseOnStandardDiffColorArea[k - 1].LineNumber + 1)
									{
										documentLine = this.textEditor2.Document.GetLineByNumber(baseOnStandardDiffColorArea[k].LineNumber);
										break;
									}
								}
							}
						}
					}
					if (documentLine == null)
					{
						documentLine = this.textEditor2.Document.GetLineByNumber(baseOnStandardDiffColorArea[0].LineNumber);
					}
					this.textEditor2.CaretOffset = documentLine.Offset;
					this.textEditor2.ScrollTo(documentLine.LineNumber);
				}
			}
		}

		// Token: 0x060001A1 RID: 417 RVA: 0x0000E67C File Offset: 0x0000C87C
		private void CanToNextDiffLine(object sender, CanExecuteRoutedEventArgs e)
		{
			if (SettingManager.Get().GetTzpManger(this.ProgramKey) != null)
			{
				e.CanExecute = SettingManager.Get().GetTzpManger(this.ProgramKey).IsDiff && ResourceController.GetInstance().GetProgramInfo(this.ProgramKey).BaseOnStandardDiffColorArea.Count<DiffColor>() > 0;
				return;
			}
			e.CanExecute = false;
		}

		// Token: 0x060001A2 RID: 418 RVA: 0x0000E700 File Offset: 0x0000C900
		private void OnToPreviousDiffLine(object sender, ExecutedRoutedEventArgs e)
		{
			if (ResourceController.GetInstance().GetProgramInfo(this.ProgramKey).BaseOnStandardDiffColorArea != null)
			{
				DocumentLine CurrentLine = this.textEditor2.Document.GetLineByOffset(this.textEditor2.CaretOffset);
				DocumentLine documentLine = null;
				List<DiffColor> baseOnStandardDiffColorArea = ResourceController.GetInstance().GetProgramInfo(this.ProgramKey).BaseOnStandardDiffColorArea;
				if (baseOnStandardDiffColorArea.Count<DiffColor>() > 0)
				{
					if (ResourceController.GetInstance().GetProgramInfo(this.ProgramKey).BaseOnStandardDiffColorArea.Where<DiffColor>((DiffColor a) => a.LineNumber == CurrentLine.LineNumber).FirstOrDefault<DiffColor>() == null)
					{
						for (int i = baseOnStandardDiffColorArea.Count<DiffColor>() - 1; i >= 0; i--)
						{
							if (baseOnStandardDiffColorArea[i].LineNumber < CurrentLine.LineNumber)
							{
								documentLine = this.textEditor2.Document.GetLineByNumber(baseOnStandardDiffColorArea[i].LineNumber);
								break;
							}
						}
					}
					else
					{
						for (int j = 0; j < baseOnStandardDiffColorArea.Count<DiffColor>(); j++)
						{
							if (baseOnStandardDiffColorArea[j].LineNumber == CurrentLine.LineNumber)
							{
								for (int k = j; k > 0; k--)
								{
									if (baseOnStandardDiffColorArea[k].LineNumber - 1 != baseOnStandardDiffColorArea[k - 1].LineNumber)
									{
										documentLine = this.textEditor2.Document.GetLineByNumber(baseOnStandardDiffColorArea[k - 1].LineNumber);
										break;
									}
								}
							}
						}
					}
					if (documentLine == null)
					{
						documentLine = this.textEditor2.Document.GetLineByNumber(baseOnStandardDiffColorArea[baseOnStandardDiffColorArea.Count<DiffColor>() - 1].LineNumber);
					}
					this.textEditor2.CaretOffset = documentLine.Offset;
					this.textEditor2.ScrollTo(documentLine.LineNumber);
				}
			}
		}

		// Token: 0x060001A3 RID: 419 RVA: 0x0000E8D0 File Offset: 0x0000CAD0
		private void CanToPreviousDiffLine(object sender, CanExecuteRoutedEventArgs e)
		{
			if (SettingManager.Get().GetTzpManger(this.ProgramKey) != null)
			{
				e.CanExecute = SettingManager.Get().GetTzpManger(this.ProgramKey).IsDiff && ResourceController.GetInstance().GetProgramInfo(this.ProgramKey).BaseOnStandardDiffColorArea.Count<DiffColor>() > 0;
				return;
			}
			e.CanExecute = false;
		}

		// Token: 0x060001A4 RID: 420 RVA: 0x0000E94C File Offset: 0x0000CB4C
		private void OnBaseOnStandardSave(object sender, ExecutedRoutedEventArgs e)
		{
			IEnumerable<AddPointModel> enumerable = ResourceController.GetInstance().GetProgramInfo(this.ProgramKey).AddPoints.Where<AddPointModel>((AddPointModel adp) => adp.Name == this.currentAPT.Name);
			for (int i = 0; i < enumerable.Count<AddPointModel>(); i++)
			{
				AddPointModel addPointModel = enumerable.ElementAt<AddPointModel>(i);
				string text = string.Empty;
				if (addPointModel.IsSelfDefinition)
				{
					text = string.Concat(new object[]
					{
						addPointModel.Scope,
						" ",
						addPointModel.Type,
						" ",
						addPointModel.FunctionName,
						"\r\n",
						this.textEditor2.Text.Replace("\a\r\n", "").Replace("\a", ""),
						"\r\nEND ",
						addPointModel.Type
					});
					addPointModel.Content = text;
				}
				else
				{
					addPointModel.Content = this.textEditor2.Text.Replace("\a\r\n", "").Replace("\a", "");
				}
				addPointModel.DiffBaseOnStandardModify = "Y";
				EventAggregatorManager.Global.GetEvent<SaveDiffBaseOnStandardResultEvent>().Publish(addPointModel.Name);
			}
			Application.Current.Exit -= this.Current_Exit;
			base.Close();
		}

		// Token: 0x060001A5 RID: 421 RVA: 0x0000EABD File Offset: 0x0000CCBD
		private void CanBaseOnStandardSave(object sender, CanExecuteRoutedEventArgs e)
		{
			e.CanExecute = true;
		}

		// Token: 0x060001A6 RID: 422 RVA: 0x0000EAC8 File Offset: 0x0000CCC8
		public void OnDiffBlockCopy(DiffCopyInfo info)
		{
			if (info.ProgramKey.Program != this.ProgramKey.Program)
			{
				return;
			}
			if (!info.IsDiffBaseOnStandard)
			{
				return;
			}
			this.textEditor2.TextArea.Document.UndoStack.StartUndoGroup();
			int offset = this.textEditor2.TextArea.Document.GetLineByNumber(info.startLine).Offset;
			int endOffset = this.textEditor2.TextArea.Document.GetLineByNumber(info.endLine).EndOffset;
			this.textEditor2.TextArea.Selection = Selection.Create(this.textEditor2.TextArea, offset, endOffset);
			this.textEditor2.TextArea.Selection.ReplaceSelectionWithText(info.SelectContent);
			this.textEditor2.TextArea.Document.UndoStack.EndUndoGroup();
		}

		// Token: 0x060001A7 RID: 423 RVA: 0x0000EBB0 File Offset: 0x0000CDB0
		public void OnDiffLineCopy(DiffCopyInfo info)
		{
			if (info.ProgramKey.Program != this.ProgramKey.Program)
			{
				return;
			}
			if (!info.IsDiffBaseOnStandard)
			{
				return;
			}
			this.textEditor2.TextArea.Document.UndoStack.StartUndoGroup();
			int offset = this.textEditor2.TextArea.Document.GetLineByNumber(info.startLine).Offset;
			int endOffset = this.textEditor2.TextArea.Document.GetLineByNumber(info.endLine).EndOffset;
			this.textEditor2.CaretOffset = offset;
			this.textEditor2.TextArea.Selection = Selection.Create(this.textEditor2.TextArea, offset, endOffset);
			this.textEditor2.TextArea.Selection.ReplaceSelectionWithText(info.SelectContent);
			this.textEditor2.TextArea.Document.UndoStack.EndUndoGroup();
		}

		// Token: 0x060001A8 RID: 424 RVA: 0x0000ECA3 File Offset: 0x0000CEA3
		private void OnRefreshDiffResult(object target, ExecutedRoutedEventArgs args)
		{
		}

		// Token: 0x060001A9 RID: 425 RVA: 0x0000ECA8 File Offset: 0x0000CEA8
		private void button_Click(object sender, RoutedEventArgs e)
		{
			this._diffManager = new DiffManager(this.textEditor1, this.textEditor1.Text.Replace("\a\r\n", ""), this.textEditor2, this.textEditor2.Text.Replace("\a\r\n", ""), true, this.ProgramKey);
		}

		// Token: 0x040000B0 RID: 176
		private static DiffBaseOnStandardWindow _this;

		// Token: 0x040000B1 RID: 177
		private DiffManager _diffManager;
	}
}
