using System;
using System.Collections.Generic;
using System.Linq;
using System.Text.RegularExpressions;
using System.Windows;
using System.Windows.Controls;
using System.Windows.Input;
using DifferenceEngine;
using ICSharpCode.AvalonEdit;
using ICSharpCode.AvalonEdit.Document;
using ICSharpCode.AvalonEdit.Editing;
using SpecDesigner.CodeEditWindow.Helper;
using SpecDesigner.Infrastructure;
using SpecDesigner.Infrastructure.Event;
using SpecDesigner.Infrastructure.Model;
using SpecDesignerCommon;
using SpecDesignerCommon.Events;

namespace SpecDesigner.CodeEditWindow.View
{
	// Token: 0x02000007 RID: 7
	public class DiffTextViewer : BaseTextEditor
	{
		// Token: 0x0600002D RID: 45 RVA: 0x00003B90 File Offset: 0x00001D90
		public DiffTextViewer()
		{
			EventAggregatorManager.Global.GetEvent<ReplaceKeywordEvent>().Unsubscribe(new Action<SearchKeywordEventArgs>(base.SearchKeyword));
			EventAggregatorManager.Global.GetEvent<ReplaceResultEvent>().Unsubscribe(new Action<ReplaceResultInfo>(base.ReplaceResultInfoSelected));
			EventAggregatorManager.Global.GetEvent<ReplaceEvent>().Unsubscribe(new Action<ReplaceResultInfo>(base.ReplaceOne));
			EventAggregatorManager.Global.GetEvent<ReplaceAllEvent>().Unsubscribe(new Action<ReplaceResultInfo>(base.ReplaceAll));
			EventAggregatorManager.Global.GetEvent<ReplaceAllFromCaret>().Unsubscribe(new Action<ReplaceResultInfo>(base.ReplaceAllFromCaret));
			EventAggregatorManager.Global.GetEvent<DiffNormalizationSearchEvent>().Subscribe(new Action<SearchKeywordEventArgs>(this.NormalizationSearch));
			base.AddHandler(FrameworkElement.ContextMenuOpeningEvent, new ContextMenuEventHandler(this.DiffTextViewer_ContextMenuOpening));
			this.IsChangedMode = false;
			this._diffRenderer = new DiffRenderer(this);
			this.DiffCopy = new List<DiffColor>();
			base.TextArea.TextView.BackgroundRenderers.Add(this._diffRenderer);
			base.Mode = ContentType.READONLY;
			this.CreateCommnad();
		}

		// Token: 0x0600002E RID: 46 RVA: 0x00003CA4 File Offset: 0x00001EA4
		public DiffTextViewer(PackageKey key)
			: this()
		{
			base.ProgramKey = new PackageKey(key.Program, key.PackType);
			base.ProgramKey.Memo = MemoType.CodeDiff;
			base.SearchProgramKey = new PackageKey(key.Program, key.PackType);
		}

		// Token: 0x0600002F RID: 47 RVA: 0x00003CF1 File Offset: 0x00001EF1
		public void AddRenderRegion(int lineNumber, DiffResultSpanStatus type)
		{
			if (this._diffRenderer != null)
			{
				this._diffRenderer.AddRenderRegion(lineNumber, type);
			}
		}

		// Token: 0x06000030 RID: 48 RVA: 0x00003D08 File Offset: 0x00001F08
		public void ClearRenderRegion()
		{
			this._diffRenderer.Clear();
		}

		// Token: 0x06000031 RID: 49 RVA: 0x00003D15 File Offset: 0x00001F15
		public new void Dispose()
		{
			base.Dispose();
			EventAggregatorManager.Global.GetEvent<DiffNormalizationSearchEvent>().Unsubscribe(new Action<SearchKeywordEventArgs>(this.NormalizationSearch));
		}

		// Token: 0x17000006 RID: 6
		// (get) Token: 0x06000032 RID: 50 RVA: 0x00003D38 File Offset: 0x00001F38
		// (set) Token: 0x06000033 RID: 51 RVA: 0x00003D40 File Offset: 0x00001F40
		public bool IsChangedMode { get; set; }

		// Token: 0x17000007 RID: 7
		// (get) Token: 0x06000034 RID: 52 RVA: 0x00003D49 File Offset: 0x00001F49
		// (set) Token: 0x06000035 RID: 53 RVA: 0x00003D51 File Offset: 0x00001F51
		public bool IsDiffBaseOnStandard { get; set; }

		// Token: 0x06000036 RID: 54 RVA: 0x00003D5A File Offset: 0x00001F5A
		private void DiffTextViewer_ContextMenuOpening(object sender, ContextMenuEventArgs e)
		{
			this.CreateContextMenu();
		}

		// Token: 0x06000037 RID: 55 RVA: 0x00003D64 File Offset: 0x00001F64
		private void CreateContextMenu()
		{
			if (base.ContextMenu != null && base.ContextMenu.Items.Count > 0)
			{
				return;
			}
			base.ContextMenu = new ContextMenu();
			base.ContextMenu.Items.Add(new MenuItem
			{
				Header = (Application.Current.FindResource("CE_DiffBlockCopyContent") as string),
				Command = CodeEditCommands.DiffBlockCopyContent
			});
			base.ContextMenu.Items.Add(new MenuItem
			{
				Header = (Application.Current.FindResource("CE_DiffLineCopyContent") as string),
				Command = CodeEditCommands.DiffLineCopyContent
			});
			base.ContextMenu.Items.Add(new MenuItem
			{
				Header = (Application.Current.FindResource("CE_DiffCopyNewFunction") as string),
				Command = CodeEditCommands.DiffFunctionSync
			});
		}

		// Token: 0x06000038 RID: 56 RVA: 0x00003E50 File Offset: 0x00002050
		private void CreateCommnad()
		{
			base.CommandBindings.Add(new CommandBinding(CodeEditCommands.DiffBlockCopyContent, new ExecutedRoutedEventHandler(DiffTextViewer.OnDiffBlockCopyContent), new CanExecuteRoutedEventHandler(DiffTextViewer.CanDiffBlockCopyContent)));
			base.CommandBindings.Add(new CommandBinding(CodeEditCommands.DiffLineCopyContent, new ExecutedRoutedEventHandler(DiffTextViewer.OnDiffLineCopyContent), new CanExecuteRoutedEventHandler(DiffTextViewer.CanDiffLineCopyContent)));
			base.CommandBindings.Add(new CommandBinding(CodeEditCommands.DiffFunctionSync, new ExecutedRoutedEventHandler(DiffTextViewer.OnDiffFunctionSync), new CanExecuteRoutedEventHandler(DiffTextViewer.CanDiffFunctionSync)));
		}

		// Token: 0x06000039 RID: 57 RVA: 0x00003F04 File Offset: 0x00002104
		private static void CanDiffBlockCopyContent(object sender, CanExecuteRoutedEventArgs e)
		{
			e.Handled = true;
			DiffTextViewer diffTextViewer = sender as DiffTextViewer;
			DocumentLine currentline = diffTextViewer.TextArea.Document.GetLineByOffset(diffTextViewer.TextArea.Caret.Offset);
			if (diffTextViewer.DiffCopy.Where<DiffColor>((DiffColor a) => a.LineNumber == currentline.LineNumber).ElementAtOrDefault<DiffColor>(0) == null)
			{
				e.CanExecute = false;
				return;
			}
			e.CanExecute = true;
		}

		// Token: 0x0600003A RID: 58 RVA: 0x00003F7C File Offset: 0x0000217C
		private static void OnDiffBlockCopyContent(object target, ExecutedRoutedEventArgs args)
		{
			DiffTextViewer diffTextViewer = target as DiffTextViewer;
			List<DiffColor> diffCopy = diffTextViewer.DiffCopy;
			int offset = diffTextViewer.TextArea.Caret.Offset;
			bool flag = false;
			bool flag2 = false;
			DocumentLine lineByOffset = diffTextViewer.TextArea.Document.GetLineByOffset(offset);
			int num = lineByOffset.LineNumber;
			int num2 = lineByOffset.LineNumber;
			if (num > 1)
			{
				for (int i = diffCopy.Count - 1; i >= 0; i--)
				{
					if (diffCopy[i].LineNumber == num)
					{
						num--;
					}
					else if (num != lineByOffset.LineNumber)
					{
						flag = true;
						num++;
						break;
					}
				}
				if (!flag)
				{
					num++;
				}
			}
			if (num2 < diffTextViewer.LineCount)
			{
				for (int j = 0; j < diffCopy.Count; j++)
				{
					if (diffCopy[j].LineNumber == num2)
					{
						num2++;
					}
					else if (num2 != lineByOffset.LineNumber)
					{
						flag2 = true;
						num2--;
						break;
					}
				}
				if (!flag2)
				{
					num2--;
				}
			}
			if (num > diffTextViewer.TextArea.Document.LineCount || num2 > diffTextViewer.TextArea.Document.LineCount)
			{
				MessageBox.Show(Application.Current.FindResource("Message_DiffCopyUnEditable") as string);
				return;
			}
			DocumentLine lineByNumber = diffTextViewer.TextArea.Document.GetLineByNumber(num);
			DocumentLine lineByNumber2 = diffTextViewer.TextArea.Document.GetLineByNumber(num2);
			diffTextViewer.TextArea.Selection = Selection.Create(diffTextViewer.TextArea, lineByNumber.Offset, lineByNumber2.EndOffset);
			string text = string.Empty;
			string[] array = DiffTextViewer.CanFunctionDiffCopy(diffTextViewer).Split(new char[] { '|' });
			text = array[1];
			DiffCopyInfo diffCopyInfo = new DiffCopyInfo();
			diffCopyInfo.ProgramKey = diffTextViewer.ProgramKey;
			diffCopyInfo.SelectContent = diffTextViewer.TextArea.Selection.GetText();
			diffCopyInfo.lineNumber = diffTextViewer.TextArea.Document.GetLineByOffset(diffTextViewer.TextArea.Caret.Offset).LineNumber;
			diffCopyInfo.IsDiffBaseOnStandard = diffTextViewer.IsDiffBaseOnStandard;
			int line = diffTextViewer.TextArea.Selection.StartPosition.Line;
			int line2 = diffTextViewer.TextArea.Selection.EndPosition.Line;
			if (line < line2)
			{
				diffCopyInfo.startLine = line;
				diffCopyInfo.endLine = line2;
			}
			else
			{
				diffCopyInfo.startLine = line2;
				diffCopyInfo.endLine = line;
			}
			diffCopyInfo.FunctionName = (Convert.ToBoolean(array[0]) ? text : "");
			EventAggregatorManager.Global.GetEvent<DiffBlockCopyContentEvent>().Publish(diffCopyInfo);
		}

		// Token: 0x0600003B RID: 59 RVA: 0x00004244 File Offset: 0x00002444
		private static void CanDiffLineCopyContent(object sender, CanExecuteRoutedEventArgs e)
		{
			e.Handled = true;
			DiffTextViewer diffTextViewer = sender as DiffTextViewer;
			DocumentLine currentline = diffTextViewer.TextArea.Document.GetLineByOffset(diffTextViewer.TextArea.Caret.Offset);
			if (diffTextViewer.DiffCopy.Where<DiffColor>((DiffColor a) => a.LineNumber == currentline.LineNumber).ElementAtOrDefault<DiffColor>(0) == null)
			{
				e.CanExecute = false;
				return;
			}
			e.CanExecute = true;
		}

		// Token: 0x0600003C RID: 60 RVA: 0x000042BC File Offset: 0x000024BC
		private static void OnDiffLineCopyContent(object target, ExecutedRoutedEventArgs args)
		{
			DiffTextViewer diffTextViewer = target as DiffTextViewer;
			string text = string.Empty;
			string[] array = DiffTextViewer.CanFunctionDiffCopy(diffTextViewer).Split(new char[] { '|' });
			text = array[1];
			DocumentLine lineByOffset = diffTextViewer.Document.GetLineByOffset(diffTextViewer.TextArea.Caret.Offset);
			diffTextViewer.TextArea.Selection = Selection.Create(diffTextViewer.TextArea, lineByOffset.Offset, lineByOffset.EndOffset);
			DiffCopyInfo diffCopyInfo = new DiffCopyInfo();
			diffCopyInfo.ProgramKey = diffTextViewer.ProgramKey;
			diffCopyInfo.SelectContent = diffTextViewer.TextArea.Selection.GetText();
			diffCopyInfo.lineNumber = diffTextViewer.TextArea.Document.GetLineByOffset(diffTextViewer.TextArea.Caret.Offset).LineNumber;
			diffCopyInfo.startLine = diffTextViewer.TextArea.Selection.StartPosition.Line;
			diffCopyInfo.endLine = diffTextViewer.TextArea.Selection.EndPosition.Line;
			diffCopyInfo.IsDiffBaseOnStandard = diffTextViewer.IsDiffBaseOnStandard;
			diffCopyInfo.FunctionName = (Convert.ToBoolean(array[0]) ? text : "");
			EventAggregatorManager.Global.GetEvent<DiffLineCopyContentEvent>().Publish(diffCopyInfo);
		}

		// Token: 0x0600003D RID: 61 RVA: 0x00004418 File Offset: 0x00002618
		private static bool CanFunctionSync(DiffTextViewer editor)
		{
			PackageKey programKey = editor.ProgramKey;
			programKey.Memo = MemoType.None;
			int num = 0;
			bool flag = true;
			AddPointModel addPointModel = ResourceController.GetInstance().GetProgramInfo(programKey).AddPoints.Where<AddPointModel>((AddPointModel a) => a.Type == DefinitionType.FUNCTION && a.SortIndex == 1).ElementAtOrDefault<AddPointModel>(0);
			int num2 = addPointModel.Description.Split(new char[] { '\n' }).Count<string>();
			string text = addPointModel.Type + " " + addPointModel.FunctionName;
			int caretOffset = editor.CaretOffset;
			int offset = editor.Document.GetOffset(editor.Document.GetLocation(caretOffset));
			for (int i = 0; i <= editor.Document.Lines.Count; i++)
			{
				DocumentLine documentLine = editor.Document.Lines[i];
				string text2 = editor.Document.GetText(documentLine.Offset, documentLine.EndOffset - documentLine.Offset);
				if (text2.Contains(text))
				{
					DocumentLine documentLine2 = editor.Document.Lines[i - num2];
					num = documentLine2.Offset;
					break;
				}
			}
			if (num == 0)
			{
				flag = false;
			}
			if (num > 0 && offset < num)
			{
				flag = false;
			}
			return flag;
		}

		// Token: 0x0600003E RID: 62 RVA: 0x00004564 File Offset: 0x00002764
		private static string CanFunctionDiffCopy(DiffTextViewer editor)
		{
			bool flag = true;
			string text = string.Empty;
			string text2 = string.Empty;
			string text3 = string.Empty;
			DocumentLine lineByOffset = editor.TextArea.Document.GetLineByOffset(editor.TextArea.Caret.Offset);
			DocumentLine documentLine = lineByOffset;
			for (int i = lineByOffset.LineNumber; i > 0; i--)
			{
				if (i <= 1)
				{
					flag = false;
				}
				else
				{
					text3 = editor.TextArea.Document.GetText(documentLine.Offset, documentLine.Length);
					if (text3.Contains("END FUNCTION"))
					{
						documentLine = editor.TextArea.Document.GetLineByNumber(documentLine.LineNumber - 1);
					}
					if (text3.Contains("PRIVATE FUNCTION") || text3.Contains("PUBLIC FUNCTION"))
					{
						break;
					}
					documentLine = editor.TextArea.Document.GetLineByOffset(documentLine.Offset - 1);
				}
			}
			if (DiffTextViewer.funcReg.IsMatch(text3))
			{
				MatchCollection matchCollection = DiffTextViewer.funcReg.Matches(text3);
				text = matchCollection[0].Groups["name"].Value;
				text2 = DiffTextViewer.noSpaceRegex.Replace(matchCollection[0].Groups["parameter"].Value, "");
			}
			return string.Concat(new string[]
			{
				flag.ToString(),
				"|",
				text,
				"|",
				text2
			});
		}

		// Token: 0x0600003F RID: 63 RVA: 0x00004704 File Offset: 0x00002904
		private static void CanDiffFunctionSync(object sender, CanExecuteRoutedEventArgs e)
		{
			e.Handled = true;
			DiffTextViewer diffTextViewer = sender as DiffTextViewer;
			DocumentLine currentline = diffTextViewer.TextArea.Document.GetLineByOffset(diffTextViewer.TextArea.Caret.Offset);
			if (diffTextViewer.DiffCopy.Where<DiffColor>((DiffColor a) => a.LineNumber == currentline.LineNumber).ElementAtOrDefault<DiffColor>(0) == null)
			{
				e.CanExecute = false;
				return;
			}
			string text = string.Empty;
			string text2 = string.Empty;
			string[] array = DiffTextViewer.CanFunctionDiffCopy(diffTextViewer).Split(new char[] { '|' });
			if (!Convert.ToBoolean(array[0]))
			{
				e.CanExecute = false;
				return;
			}
			text = array[1];
			text2 = array[0];
			DiffTextViewer._functionSyncinfo = new FunctionSyncInfo();
			DiffTextViewer._functionSyncinfo.FunctionName = text;
			DiffTextViewer._functionSyncinfo.parameter = text2;
			e.CanExecute = true;
		}

		// Token: 0x06000040 RID: 64 RVA: 0x000047E4 File Offset: 0x000029E4
		private static void OnDiffFunctionSync(object target, ExecutedRoutedEventArgs args)
		{
			if (DiffTextViewer._functionSyncinfo.TapSrcIsNull)
			{
				MessageBox.Show(Application.Current.FindResource("Message_TapSrcIsNull") as string);
				return;
			}
			FunctionSyncInfo functionSyncInfo = new FunctionSyncInfo();
			functionSyncInfo.FullFunction = DiffTextViewer._functionSyncinfo.FullFunction;
			functionSyncInfo.FunctionName = DiffTextViewer._functionSyncinfo.FunctionName;
			EventAggregatorManager.Global.GetEvent<DiffFunctionSyncEvent>().Publish(functionSyncInfo);
		}

		// Token: 0x06000041 RID: 65 RVA: 0x00004850 File Offset: 0x00002A50
		public void NormalizationSearch(SearchKeywordEventArgs e)
		{
			if (e.Type != DiffType.Diff)
			{
				return;
			}
			if (base.ProgramKey == null)
			{
				base.ProgramKey = e.ProgramKey;
			}
			if (base.SearchProgramKey != e.ProgramKey)
			{
				base.SearchProgramKey = e.ProgramKey;
			}
			base.SearchKeyword(e, DiffType.Diff);
		}

		// Token: 0x06000042 RID: 66 RVA: 0x000048A8 File Offset: 0x00002AA8
		public void DiffTextViewerAddNewLine(AddNewLine line)
		{
			base.Mode = ContentType.OPEN;
			DocumentLine lineByNumber = base.TextArea.Document.GetLineByNumber(line.LineNumber);
			base.TextArea.Selection = Selection.Create(base.TextArea, lineByNumber.Offset, lineByNumber.EndOffset);
			string text = "\r\n" + base.TextArea.Selection.GetText();
			base.TextArea.Caret.Offset = lineByNumber.Offset;
			base.TextArea.Selection.ReplaceSelectionWithText(text);
			base.Mode = ContentType.READONLY;
			if (!this.IsChangedMode)
			{
				this.IsChangedMode = true;
			}
		}

		// Token: 0x06000043 RID: 67 RVA: 0x00004950 File Offset: 0x00002B50
		public void DiffTextViewerCtrlZ(AddNewLine line)
		{
			base.Mode = ContentType.OPEN;
			DocumentLine lineByNumber = base.TextArea.Document.GetLineByNumber(line.LineNumber);
			string text = base.Document.GetText(lineByNumber.Offset, lineByNumber.EndOffset - lineByNumber.Offset);
			if (text == "\a\r\n")
			{
				base.Document.UndoStack.Undo();
			}
			base.Mode = ContentType.READONLY;
			if (!this.IsChangedMode)
			{
				this.IsChangedMode = true;
			}
		}

		// Token: 0x06000044 RID: 68 RVA: 0x000049D0 File Offset: 0x00002BD0
		public void DiffTextViewerDeleteLine(AddNewLine line)
		{
			DocumentLine lineByNumber = base.TextArea.Document.GetLineByNumber(line.LineNumber);
			base.TextArea.Selection = Selection.Create(base.TextArea, lineByNumber.Offset, lineByNumber.EndOffset);
			string text = base.TextArea.Selection.GetText();
			if (text == "")
			{
				DocumentLine lineByNumber2 = base.TextArea.Document.GetLineByNumber(line.LineNumber - 1);
				base.TextArea.Selection = Selection.Create(base.TextArea, lineByNumber2.Offset, lineByNumber.EndOffset);
				text = base.TextArea.Selection.GetText();
				if (text == "\r\n")
				{
					base.Mode = ContentType.OPEN;
					base.TextArea.Caret.Offset = lineByNumber2.Offset;
					base.TextArea.Selection.ReplaceSelectionWithText("");
					base.Mode = ContentType.READONLY;
					if (!this.IsChangedMode)
					{
						this.IsChangedMode = true;
					}
				}
			}
		}

		// Token: 0x04000016 RID: 22
		private DiffRenderer _diffRenderer;

		// Token: 0x04000017 RID: 23
		public List<DiffColor> DiffCopy;

		// Token: 0x04000018 RID: 24
		private static RegexOptions regexOptions = RegexOptions.IgnoreCase | RegexOptions.Singleline;

		// Token: 0x04000019 RID: 25
		private static Regex funcReg = new Regex("^\\s*(private function|public function)+\\s+(?<name>\\w+)*(?<parameter>\\([^\\)]*\\))(\\r{0,1})(\\n{0,1})", DiffTextViewer.regexOptions);

		// Token: 0x0400001A RID: 26
		private static Regex noSpaceRegex = new Regex("\\s+", DiffTextViewer.regexOptions);

		// Token: 0x0400001B RID: 27
		private static FunctionSyncInfo _functionSyncinfo = null;
	}
}
