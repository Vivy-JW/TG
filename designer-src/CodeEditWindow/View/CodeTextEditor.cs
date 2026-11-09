using System;
using System.Collections;
using System.Collections.Generic;
using System.Collections.ObjectModel;
using System.ComponentModel;
using System.IO;
using System.Linq;
using System.Text;
using System.Text.RegularExpressions;
using System.Windows;
using System.Windows.Controls;
using System.Windows.Controls.Primitives;
using System.Windows.Data;
using System.Windows.Documents;
using System.Windows.Input;
using System.Windows.Interactivity;
using System.Windows.Resources;
using System.Windows.Threading;
using System.Xml;
using System.Xml.Linq;
using DifferenceEngine;
using ICSharpCode.AvalonEdit;
using ICSharpCode.AvalonEdit.Bookmark;
using ICSharpCode.AvalonEdit.CodeCompletion;
using ICSharpCode.AvalonEdit.Document;
using ICSharpCode.AvalonEdit.Editing;
using ICSharpCode.AvalonEdit.Extension;
using ICSharpCode.AvalonEdit.Highlighting;
using ICSharpCode.AvalonEdit.Highlighting.Xshd;
using ICSharpCode.AvalonEdit.Rendering;
using SpecDesigner.CodeEditWindow.Helper;
using SpecDesigner.CodeEditWindow.Model;
using SpecDesigner.Controls.Controls;
using SpecDesigner.Infrastructure;
using SpecDesigner.Infrastructure.Event;
using SpecDesigner.Infrastructure.Helper;
using SpecDesigner.Infrastructure.Model;
using SpecDesignerCommon;
using SpecDesignerCommon.Bookmark;
using SpecDesignerCommon.Connection;
using SpecDesignerCommon.Events;
using SpecDesignerCommon.Helpers;
using SpecDesignerPreference;

namespace SpecDesigner.CodeEditWindow.View
{
	// Token: 0x02000015 RID: 21
	public class CodeTextEditor : BaseTextEditor, IDisposable
	{
		// Token: 0x17000014 RID: 20
		// (get) Token: 0x06000090 RID: 144 RVA: 0x00005BCA File Offset: 0x00003DCA
		// (set) Token: 0x06000091 RID: 145 RVA: 0x00005BD2 File Offset: 0x00003DD2
		public bool IsContentLoaded
		{
			get
			{
				return this._isContentLoaded;
			}
			set
			{
				this._isContentLoaded = value;
			}
		}

		// Token: 0x06000092 RID: 146 RVA: 0x00005BDC File Offset: 0x00003DDC
		public CodeTextEditor(PackageKey key)
			: this()
		{
			base.ProgramKey = key;
			base.SearchProgramKey = key;
			Interaction.GetBehaviors(this).Add(new BookmarkTrackerBehavior());
			if (SettingManager.Get().GetTzpManger(base.ProgramKey).IsDiff)
			{
				Interaction.GetBehaviors(this).Add(new DiffTrackerBehavior());
			}
		}

		// Token: 0x06000093 RID: 147 RVA: 0x00005C34 File Offset: 0x00003E34
		private void OnInsertCodeEvent(InsertionCodeEventArgs e)
		{
			if (e.ProgramKey != base.ProgramKey)
			{
				return;
			}
			if (CodeTextEditor.IsEditable(this))
			{
				base.TextArea.PerformTextInput(e.Content);
				return;
			}
			DesignerMessageBox.Show(Application.Current.FindResource("Message_CantInput") as string, Application.Current.FindResource("Message_Message") as string);
		}

		// Token: 0x06000094 RID: 148 RVA: 0x00005CA0 File Offset: 0x00003EA0
		public CodeTextEditor()
		{
			this.InitializeCodeCompletion();
			if (DesignerProperties.GetIsInDesignMode(this))
			{
				return;
			}
			Binding binding = new Binding("Theme")
			{
				Source = PreferenceManager.Current.Settings,
				UpdateSourceTrigger = UpdateSourceTrigger.PropertyChanged,
				Mode = BindingMode.TwoWay
			};
			base.SetBinding(CodeTextEditor.ThemeProperty, binding);
			this.LoadSyntaxHighlighting();
			base.AddHandler(UIElement.PreviewMouseWheelEvent, new MouseWheelEventHandler(this.OnMouseWheelChanged));
			base.AddHandler(FrameworkElement.ContextMenuOpeningEvent, new ContextMenuEventHandler(this.CodeTextEditor_ContextMenuOpening));
			this._markedRenderer = new TextMarkedRenderer();
			base.TextArea.TextView.BackgroundRenderers.Add(this._markedRenderer);
			CaretLineRenderer caretLineRenderer = new CaretLineRenderer(this);
			base.TextArea.TextView.BackgroundRenderers.Add(caretLineRenderer);
			this._diffRenderer = new DiffRenderer(this);
			base.TextArea.TextView.BackgroundRenderers.Add(this._diffRenderer);
			ObservableCollection<UIElement> leftMargins = base.TextArea.LeftMargins;
			BookmarkBarManager bookmarkBarManager = new BookmarkBarManager();
			base.TextArea.TextView.Services.AddService(typeof(IBookmarkMargin), bookmarkBarManager);
			leftMargins.Insert(2, new BookmarkBarMargin(bookmarkBarManager));
			base.TextArea.TextEntering += this.TextAreaTextEntering;
			this.CreateCommnad();
			base.TextArea.TextView.LineTransformers.Add(new NonHighlightingColorizer());
			EventAggregatorManager.Global.GetEvent<LoadSpecReferFilesEvent>().Subscribe(new Action<string>(this.OnBasicDataUpdate));
			EventAggregatorManager.Global.GetEvent<InsertionCodeEvent>().Subscribe(new Action<InsertionCodeEventArgs>(this.OnInsertCodeEvent));
			EventAggregatorManager.Global.GetEvent<DiffBlockCopyContentEvent>().Subscribe(new Action<DiffCopyInfo>(this.OnDiffBlockCopy));
			EventAggregatorManager.Global.GetEvent<DiffLineCopyContentEvent>().Subscribe(new Action<DiffCopyInfo>(this.OnDiffLineCopy));
			EventAggregatorManager.Global.GetEvent<DiffFunctionSyncEvent>().Subscribe(new Action<FunctionSyncInfo>(this.OnDiffFunctionSync));
			EventAggregatorManager.Global.GetEvent<DiffNormalizationSearchEvent>().Subscribe(new Action<SearchKeywordEventArgs>(this.NormalizationSearch));
			EventAggregatorManager.Global.GetEvent<RefreshDiffResultEvent>().Subscribe(new Action<PackageKey>(this.OnRefreshStardandDiffResult));
			EventAggregatorManager.Global.GetEvent<SaveDiffBaseOnStandardResultEvent>().Subscribe(new Action<string>(this.OnSaveDiffStardandResult));
			EventAggregatorManager.Global.GetEvent<ProgramErrorCheckEvent>().Subscribe(new Action<PackageKey>(this.OnProgramErrorCheck));
			EventAggregatorManager.Global.GetEvent<SaveSqlResultEvent>().Subscribe(new Action<string>(this.OnSaveSqlResult));
		}

		// Token: 0x06000095 RID: 149 RVA: 0x00005F40 File Offset: 0x00004140
		private void TextAreaTextEntering(object sender, TextCompositionEventArgs e)
		{
			if (e.Handled)
			{
				return;
			}
			if (e.Text.Length > 1)
			{
				return;
			}
			if (!PreferenceManager.Current.Settings.CodeCompletion)
			{
				return;
			}
			if (!CodeTextEditor.IsEditable(this))
			{
				return;
			}
			if (this.CompletionWindow != null && this.CompletionWindow.IsVisible)
			{
				return;
			}
			TextArea textArea = base.TextArea;
			if (textArea.ActiveInputHandler != textArea.DefaultInputHandler)
			{
				return;
			}
			foreach (char c in e.Text)
			{
				switch (this._codeCompletion.HandleKeyPress(this, c))
				{
				case CodeCompletionKeyPressResult.Completed:
					if (this.CompletionWindow != null)
					{
						this.CompletionWindow.ExpectInsertionBeforeStart = true;
					}
					if (this.InsightWindow != null)
					{
						this.InsightWindow.ExpectInsertionBeforeStart = true;
					}
					break;
				case CodeCompletionKeyPressResult.CompletedIncludeKeyInCompletion:
					if (this.CompletionWindow != null && this.CompletionWindow.StartOffset == this.CompletionWindow.EndOffset)
					{
						this.CompletionWindow.CloseWhenCaretAtBeginning = true;
					}
					break;
				case CodeCompletionKeyPressResult.EatKey:
					e.Handled = true;
					break;
				}
			}
		}

		// Token: 0x06000096 RID: 150 RVA: 0x00006088 File Offset: 0x00004288
		public override void ShowInsightWindow()
		{
			string word = base.GetWordOrSymbolBeforeCaret();
			if (this.InsightWindow != null)
			{
				this.InsightWindow.Close();
			}
			IIntellisenseModel intellisenseModel = ResourceController.GetInstance().GeneralFunctionSuggestion.Where<IIntellisenseModel>((IIntellisenseModel m) => m.Name == word && m.IsFavorited).FirstOrDefault<IIntellisenseModel>();
			MethodInsightItem methodInsightItem = MethodInsightItem.Create(intellisenseModel);
			if (methodInsightItem != null)
			{
				this.InsightWindow = new DesignerInsightWindow(base.TextArea, methodInsightItem);
				this.InsightWindow.Closed += this.InsightWindow_Closed;
				this.InsightWindow.Show();
			}
		}

		// Token: 0x06000097 RID: 151 RVA: 0x00006119 File Offset: 0x00004319
		private void InsightWindow_Closed(object sender, EventArgs e)
		{
			this.InsightWindow.Closed -= this.InsightWindow_Closed;
			GC.Collect();
		}

		// Token: 0x06000098 RID: 152 RVA: 0x0000614C File Offset: 0x0000434C
		private void ShowColumnCompletionWindow(string beforeWord, IEnumerable<ICompletionData> dataSource)
		{
			if (this.CompletionWindow != null)
			{
				this.CompletionWindow.Close();
			}
			this.CompletionWindow = new DesignerCompletionWindow(this, base.TextArea, dataSource);
			this.CompletionWindow.Closed += this.OnCompletionWindowClosed;
			base.Dispatcher.BeginInvoke(DispatcherPriority.Normal, new Action(delegate
			{
				if (this.CompletionWindow != null)
				{
					this.CompletionWindow.Show();
				}
			}));
		}

		// Token: 0x06000099 RID: 153 RVA: 0x000061C8 File Offset: 0x000043C8
		private void ShowCompletionWindow(bool includingBeforeWord, IEnumerable<ICompletionData> dataSource)
		{
			if (this.CompletionWindow != null)
			{
				this.CompletionWindow.Close();
			}
			if (this.GetWordUnderCaret().Length > 0 && !includingBeforeWord)
			{
				return;
			}
			this.CompletionWindow = new DesignerCompletionWindow(this, base.TextArea, dataSource);
			this.CompletionWindow.Closed += this.OnCompletionWindowClosed;
			if (includingBeforeWord)
			{
				this.CompletionWindow.IncludeWordBefore(base.GetWordOrSymbolBeforeCaret());
			}
			base.Dispatcher.BeginInvoke(DispatcherPriority.Normal, new Action(delegate
			{
				if (this.CompletionWindow != null)
				{
					this.CompletionWindow.Show();
				}
			}));
		}

		// Token: 0x0600009A RID: 154 RVA: 0x00006252 File Offset: 0x00004452
		private void OnCompletionWindowClosed(object sender, EventArgs e)
		{
			this.CompletionWindow.Closed -= this.OnCompletionWindowClosed;
			GC.Collect();
		}

		// Token: 0x0600009B RID: 155 RVA: 0x0000629C File Offset: 0x0000449C
		public override void ShowColumnCompletionWindow(string table)
		{
			XElement xelement = TableColumnHelper.FindTableColumns(table);
			if (xelement != null)
			{
				List<CodeCompletionData> list = new List<CodeCompletionData>();
				foreach (XElement xelement2 in xelement.Elements("column"))
				{
					CodeCompletionData codeCompletionData = CodeCompletionData.Create((xelement2.Attribute("name") != null) ? xelement2.Attribute("name").Value : string.Empty);
					codeCompletionData.DescriptionText = string.Format("{0}{1}( {2} )", (xelement2.Attribute("text") != null) ? xelement2.Attribute("text").Value : string.Empty, (xelement2.Attribute("text") != null) ? "  " : "", (xelement2.Attribute("type") != null) ? xelement2.Attribute("type").Value : string.Empty);
					if (codeCompletionData != null)
					{
						list.Add(codeCompletionData);
					}
				}
				this.ShowColumnCompletionWindow(table, list);
			}
			IEnumerable<XElement> enumerable = from query in SettingManager.Get().TopGlobals.Elements("var")
				where query.Attribute("name").Value == table
				select query;
			if (enumerable != null)
			{
				List<CodeCompletionData> list2 = new List<CodeCompletionData>();
				foreach (XElement xelement3 in enumerable.Elements<XElement>("element"))
				{
					CodeCompletionData codeCompletionData2 = CodeCompletionData.Create((xelement3.Attribute("name") != null) ? xelement3.Attribute("name").Value : string.Empty);
					codeCompletionData2.DescriptionText = Application.Current.FindResource("CE_ShowGlobals") as string;
					if (codeCompletionData2 != null)
					{
						list2.Add(codeCompletionData2);
					}
				}
				this.ShowColumnCompletionWindow(table, list2);
			}
		}

		// Token: 0x0600009C RID: 156 RVA: 0x000064E4 File Offset: 0x000046E4
		public override void ShowCompletionWindow()
		{
			this.ShowCompletionWindow(false, this._codeCompletion.Items);
		}

		// Token: 0x0600009D RID: 157 RVA: 0x000064F8 File Offset: 0x000046F8
		public override void HideInsightWindow()
		{
			if (this.InsightWindow != null)
			{
				this.InsightWindow.Close();
			}
		}

		// Token: 0x0600009E RID: 158 RVA: 0x0000650D File Offset: 0x0000470D
		private void OnBasicDataUpdate(string empty)
		{
			this.InitializeCodeCompletion();
		}

		// Token: 0x0600009F RID: 159 RVA: 0x00006518 File Offset: 0x00004718
		private void OnMouseWheelChanged(object sender, MouseWheelEventArgs e)
		{
			if (Keyboard.Modifiers == ModifierKeys.Control)
			{
				int delta = e.Delta;
				int num = (int)base.FontSize;
				try
				{
					if (delta > 0)
					{
						base.FontSize += 1.0;
					}
					else if (delta < 0)
					{
						base.FontSize -= 1.0;
					}
				}
				catch
				{
					base.FontSize = (double)num;
				}
			}
		}

		// Token: 0x060000A0 RID: 160 RVA: 0x00006590 File Offset: 0x00004790
		private void CodeTextEditor_ContextMenuOpening(object sender, ContextMenuEventArgs e)
		{
			this.CreateContextMenu();
		}

		// Token: 0x060000A1 RID: 161 RVA: 0x00006598 File Offset: 0x00004798
		private void CodeTextEditor_DiffPreviewKeyDown(object sender, KeyEventArgs e)
		{
			if (e.Key == Key.Return)
			{
				AddNewLine addNewLine = new AddNewLine();
				addNewLine.ProgramKey = base.ProgramKey;
				addNewLine.LineNumber = base.TextArea.Document.GetLineByOffset(base.CaretOffset).LineNumber;
				EventAggregatorManager.Global.GetEvent<DiffTextViewerAddNewLineEvent>().Publish(addNewLine);
				return;
			}
			if (e.Key == Key.Back)
			{
				DocumentLine lineByOffset = base.TextArea.Document.GetLineByOffset(base.CaretOffset);
				AddNewLine addNewLine2 = new AddNewLine();
				addNewLine2.ProgramKey = base.ProgramKey;
				addNewLine2.LineNumber = lineByOffset.LineNumber;
				if (lineByOffset.Offset == base.CaretOffset && base.TextArea.Selection.GetText() == "")
				{
					EventAggregatorManager.Global.GetEvent<DiffTextViewerDeleteLineEvent>().Publish(addNewLine2);
					return;
				}
			}
			else if (e.Key == Key.Z && (Keyboard.Modifiers & ModifierKeys.Control) == ModifierKeys.Control)
			{
				AddNewLine addNewLine3 = new AddNewLine();
				addNewLine3.ProgramKey = base.ProgramKey;
				addNewLine3.LineNumber = base.TextArea.Document.GetLineByOffset(base.CaretOffset).LineNumber;
				EventAggregatorManager.Global.GetEvent<DiffTextViewerCtrlZEvent>().Publish(addNewLine3);
			}
		}

		// Token: 0x060000A2 RID: 162 RVA: 0x000066C8 File Offset: 0x000048C8
		private CustomPopupPlacement[] placePopup(Size popupSize, Size targetSize, Point offset)
		{
			Point point;
			if (!this._isTyped)
			{
				point = Mouse.GetPosition(this);
			}
			else
			{
				point = base.TextArea.TextView.GetVisualPosition(base.TextArea.Caret.Position, VisualYPosition.LineTop);
			}
			double num = point.X + popupSize.Width;
			double num2 = point.Y + popupSize.Height;
			CustomPopupPlacement customPopupPlacement = new CustomPopupPlacement(new Point(num, num2), PopupPrimaryAxis.Horizontal);
			CustomPopupPlacement customPopupPlacement2 = new CustomPopupPlacement(new Point(-num, num2), PopupPrimaryAxis.Vertical);
			return new CustomPopupPlacement[] { customPopupPlacement, customPopupPlacement2 };
		}

		// Token: 0x060000A3 RID: 163 RVA: 0x00006770 File Offset: 0x00004970
		private void CreateCommnad()
		{
			base.CommandBindings.Add(new CommandBinding(CodeEditCommands.KeywordUpperCase, new ExecutedRoutedEventHandler(CodeTextEditor.OnKeywordUpper), new CanExecuteRoutedEventHandler(this.CanEdit)));
			base.CommandBindings.Add(new CommandBinding(CodeEditCommands.BlockComment, new ExecutedRoutedEventHandler(CodeTextEditor.OnBlockComment), new CanExecuteRoutedEventHandler(this.CanEdit)));
			base.CommandBindings.Add(new CommandBinding(CodeEditCommands.UnblockComment, new ExecutedRoutedEventHandler(CodeTextEditor.OnUnBlockComment), new CanExecuteRoutedEventHandler(this.CanEdit)));
			base.CommandBindings.Add(new CommandBinding(CodeEditCommands.CodeCompletion, new ExecutedRoutedEventHandler(CodeTextEditor.OnCodeCompletion), new CanExecuteRoutedEventHandler(this.CanEdit)));
			base.CommandBindings.Add(new CommandBinding(CodeEditCommands.FavoriteCodeCompletion, new ExecutedRoutedEventHandler(CodeTextEditor.OnFavoriteCodeCompletion), new CanExecuteRoutedEventHandler(this.CanEdit)));
			base.CommandBindings.Add(new CommandBinding(CodeEditCommands.ColumnCodeCompletion, new ExecutedRoutedEventHandler(CodeTextEditor.OnColumnCodeCompletion), new CanExecuteRoutedEventHandler(this.CanEdit)));
			base.CommandBindings.Add(new CommandBinding(CodeEditCommands.ShowFunction, new ExecutedRoutedEventHandler(CodeTextEditor.OnShowFunction), new CanExecuteRoutedEventHandler(CodeTextEditor.CanShowFunctionDetail)));
			base.CommandBindings.Add(new CommandBinding(CodeEditCommands.RefreshScreen, new ExecutedRoutedEventHandler(this.OnRefreshScreen)));
			base.CommandBindings.Add(new CommandBinding(CodeEditCommands.FramMark, new ExecutedRoutedEventHandler(CodeTextEditor.OnFramMark), new CanExecuteRoutedEventHandler(CodeTextEditor.CanFramMark)));
			base.CommandBindings.Add(new CommandBinding(CodeEditCommands.Rename, new ExecutedRoutedEventHandler(CodeTextEditor.ExecutedRename), new CanExecuteRoutedEventHandler(CodeTextEditor.CanFunctionModify)));
			base.CommandBindings.Add(new CommandBinding(CodeEditCommands.DeleteFunction, new ExecutedRoutedEventHandler(CodeEditorMainWindow.ExecutedDeleteFunction), new CanExecuteRoutedEventHandler(CodeEditorMainWindow.CanDeleteFunction)));
			base.CommandBindings.Add(new CommandBinding(CodeEditCommands.GoToLine, new ExecutedRoutedEventHandler(CodeTextEditor.ExecutedGoToLine)));
			base.CommandBindings.Add(new CommandBinding(CodeEditCommands.GoToDiffLine, new ExecutedRoutedEventHandler(CodeTextEditor.ExecutedGoToDiffLine)));
			base.CommandBindings.Add(new CommandBinding(CodeEditCommands.InsertCodeSample, new ExecutedRoutedEventHandler(CodeTextEditor.ExecutedInsertCodeSample), new CanExecuteRoutedEventHandler(CodeTextEditor.CanInsertCodeSample)));
			base.CommandBindings.Add(new CommandBinding(CodeEditCommands.CiteOrNot, new ExecutedRoutedEventHandler(CodeTextEditor.OnCiteOrNot), new CanExecuteRoutedEventHandler(CodeTextEditor.CanCiteOrNot)));
			base.CommandBindings.Add(new CommandBinding(CodeEditCommands.CopyAllColumnsToRecord, new ExecutedRoutedEventHandler(CodeTextEditor.OnCopyAllColumnsToRecord), new CanExecuteRoutedEventHandler(CodeTextEditor.CanCopyAllColumnsToRecord)));
			base.CommandBindings.Add(new CommandBinding(CodeEditCommands.CopyColumnsName, new ExecutedRoutedEventHandler(CodeTextEditor.OnCopyColumnsName), new CanExecuteRoutedEventHandler(CodeTextEditor.CanCopyColumnsName)));
			base.CommandBindings.Add(new CommandBinding(CodeEditCommands.ReturnToStandard, new ExecutedRoutedEventHandler(CodeTextEditor.OnReturnToStandard), new CanExecuteRoutedEventHandler(CodeTextEditor.CanReturnToStandard)));
			base.CommandBindings.Add(new CommandBinding(CodeEditCommands.RefreshDiffResult, new ExecutedRoutedEventHandler(this.OnRefreshDiffResult)));
			base.CommandBindings.Add(new CommandBinding(CodeEditCommands.DiffSingleContent, new ExecutedRoutedEventHandler(CodeTextEditor.OnDiffSingleContent), new CanExecuteRoutedEventHandler(CodeTextEditor.CanDiffSingleContent)));
			base.CommandBindings.Add(new CommandBinding(CodeEditCommands.DiffBaseOnStandard, new ExecutedRoutedEventHandler(CodeTextEditor.OnDiffBaseOnStandard), new CanExecuteRoutedEventHandler(CodeTextEditor.CanDiffBaseOnStandard)));
			base.CommandBindings.Add(new CommandBinding(CodeEditCommands.FullTextKeywordUpperCase, new ExecutedRoutedEventHandler(CodeTextEditor.OnFullTextKeywordUpper), new CanExecuteRoutedEventHandler(CodeTextEditor.CanFullTextKeywordUpper)));
			base.CommandBindings.Add(new CommandBinding(CodeEditCommands.GenerateSqlTemplate, new ExecutedRoutedEventHandler(CodeTextEditor.OnGenerateSqlTemplate), new CanExecuteRoutedEventHandler(CodeTextEditor.CanGenerateSqlTemplate)));
		}

		// Token: 0x060000A4 RID: 164 RVA: 0x00006B6F File Offset: 0x00004D6F
		private static TextArea GetTextArea(object target)
		{
			if (target is TextEditor)
			{
				return (target as TextEditor).TextArea;
			}
			if (target is TextArea)
			{
				return target as TextArea;
			}
			return null;
		}

		// Token: 0x060000A5 RID: 165 RVA: 0x00006B95 File Offset: 0x00004D95
		private void CanEdit(object sender, CanExecuteRoutedEventArgs e)
		{
			this._lastCheckEditable = CodeTextEditor.IsEditable(this);
			e.CanExecute = this._lastCheckEditable;
		}

		// Token: 0x060000A6 RID: 166 RVA: 0x00006BC8 File Offset: 0x00004DC8
		private static void OnKeywordUpper(object target, ExecutedRoutedEventArgs args)
		{
			TextArea textArea = CodeTextEditor.GetTextArea(target);
			if (textArea != null && textArea.Document != null)
			{
				CodeTextEditor.<>c__DisplayClassb CS$<>8__locals1 = new CodeTextEditor.<>c__DisplayClassb();
				if (textArea.Selection.IsEmpty)
				{
					return;
				}
				ISegment surroundingSegment = textArea.Selection.SurroundingSegment;
				IEnumerable<ISegment> enumerable = textArea.Selection.Segments.Cast<ISegment>();
				StringBuilder stringBuilder = new StringBuilder();
				CS$<>8__locals1.word = string.Empty;
				for (int i = 0; i < enumerable.Count<ISegment>(); i++)
				{
					ISegment segment = enumerable.ElementAtOrDefault<ISegment>(i);
					if (textArea.Document.SectionProvider.GetDeletableSegments(segment).Count<ISegment>() == 0)
					{
						return;
					}
					for (int j = segment.Offset; j <= segment.EndOffset; j++)
					{
						char charAt = textArea.Document.GetCharAt(j);
						if (!char.IsWhiteSpace(charAt))
						{
							CodeTextEditor.<>c__DisplayClassb CS$<>8__locals2 = CS$<>8__locals1;
							CS$<>8__locals2.word += charAt;
						}
						else
						{
							if (ResourceController.GetInstance().FglKeywords.Any<string>((string s) => string.Equals(s, CS$<>8__locals1.word, StringComparison.InvariantCultureIgnoreCase)))
							{
								stringBuilder.Append(CS$<>8__locals1.word.ToUpper());
							}
							else
							{
								stringBuilder.Append(CS$<>8__locals1.word);
							}
							if (segment.EndOffset != j)
							{
								stringBuilder.Append(charAt);
							}
							CS$<>8__locals1.word = string.Empty;
						}
					}
				}
				if (CS$<>8__locals1.word != string.Empty)
				{
					stringBuilder.Append(CS$<>8__locals1.word.Substring(0, CS$<>8__locals1.word.Length - 1));
				}
				textArea.Selection.ReplaceSelectionWithText(stringBuilder.ToString());
				textArea.Selection = Selection.Create(textArea, surroundingSegment.Offset, surroundingSegment.EndOffset);
			}
			textArea.Caret.BringCaretToView();
			args.Handled = true;
		}

		// Token: 0x060000A7 RID: 167 RVA: 0x00006DA4 File Offset: 0x00004FA4
		private static void OnBlockComment(object target, ExecutedRoutedEventArgs args)
		{
			TextArea textArea = CodeTextEditor.GetTextArea(target);
			if (textArea != null && textArea.Document != null)
			{
				RegexOptions regexOptions = RegexOptions.IgnoreCase | RegexOptions.Multiline;
				Regex regex = new Regex("^", regexOptions);
				if (!textArea.Selection.IsEmpty)
				{
					ISegment surroundingSegment = textArea.Selection.SurroundingSegment;
					ISegment surroundingSegment2 = textArea.Selection.SurroundingSegment;
					DocumentLine lineByOffset = textArea.Document.GetLineByOffset(surroundingSegment2.Offset);
					DocumentLine lineByOffset2 = textArea.Document.GetLineByOffset(surroundingSegment2.EndOffset);
					textArea.ClearSelection();
					textArea.Selection = Selection.Create(textArea, lineByOffset.Offset, lineByOffset2.EndOffset);
					string text = textArea.Selection.GetText();
					textArea.Selection.ReplaceSelectionWithText(regex.Replace(text, "#"));
					textArea.Selection = Selection.Create(textArea, surroundingSegment.Offset, surroundingSegment.EndOffset);
					return;
				}
				int offset = textArea.Caret.Offset;
				DocumentLine lineByOffset3 = textArea.Document.GetLineByOffset(offset);
				TextSegment textSegment = new TextSegment
				{
					StartOffset = lineByOffset3.Offset,
					EndOffset = lineByOffset3.EndOffset
				};
				string text2 = textArea.Document.GetText(textSegment);
				textArea.Document.Replace(textSegment, regex.Replace(text2, "#"));
			}
		}

		// Token: 0x060000A8 RID: 168 RVA: 0x00006EF4 File Offset: 0x000050F4
		private static void OnUnBlockComment(object target, ExecutedRoutedEventArgs args)
		{
			TextArea textArea = CodeTextEditor.GetTextArea(target);
			if (textArea != null && textArea.Document != null)
			{
				RegexOptions regexOptions = RegexOptions.IgnoreCase | RegexOptions.Multiline;
				Regex regex = new Regex("^(.{0,1})", regexOptions);
				if (!textArea.Selection.IsEmpty)
				{
					ISegment surroundingSegment = textArea.Selection.SurroundingSegment;
					DocumentLine lineByOffset = textArea.Document.GetLineByOffset(surroundingSegment.Offset);
					DocumentLine lineByOffset2 = textArea.Document.GetLineByOffset(surroundingSegment.EndOffset);
					textArea.ClearSelection();
					textArea.Selection = Selection.Create(textArea, lineByOffset.Offset, lineByOffset2.EndOffset);
					string text = textArea.Selection.GetText();
					textArea.Selection.ReplaceSelectionWithText(regex.Replace(text, ""));
					return;
				}
				int offset = textArea.Caret.Offset;
				DocumentLine lineByOffset3 = textArea.Document.GetLineByOffset(offset);
				TextSegment textSegment = new TextSegment
				{
					StartOffset = lineByOffset3.Offset,
					EndOffset = lineByOffset3.EndOffset
				};
				string text2 = textArea.Document.GetText(textSegment);
				textArea.Document.Replace(textSegment, regex.Replace(text2, ""));
			}
		}

		// Token: 0x060000A9 RID: 169 RVA: 0x00007018 File Offset: 0x00005218
		private static void OnCodeCompletion(object target, ExecutedRoutedEventArgs args)
		{
			CodeTextEditor codeTextEditor = target as CodeTextEditor;
			if (codeTextEditor == null)
			{
				return;
			}
			codeTextEditor.ShowCompletionWindow(true, codeTextEditor._codeCompletion.Items);
		}

		// Token: 0x060000AA RID: 170 RVA: 0x00007044 File Offset: 0x00005244
		private static void OnFavoriteCodeCompletion(object target, ExecutedRoutedEventArgs args)
		{
			CodeTextEditor codeTextEditor = target as CodeTextEditor;
			if (codeTextEditor == null)
			{
				return;
			}
			List<CodeCompletionData> list = new List<CodeCompletionData>();
			foreach (IIntellisenseModel intellisenseModel in ResourceController.GetInstance().GeneralFunctionSuggestion)
			{
				if (intellisenseModel.IsFavorited)
				{
					CodeCompletionData codeCompletionData = CodeCompletionData.Create(intellisenseModel);
					if (codeCompletionData != null)
					{
						list.Add(codeCompletionData);
					}
				}
			}
			if (codeTextEditor.GetWordOrSymbolBeforeCaret().Length > 0)
			{
				codeTextEditor.ShowCompletionWindow(true, list);
				return;
			}
			codeTextEditor.ShowCompletionWindow(false, list);
		}

		// Token: 0x060000AB RID: 171 RVA: 0x000070DC File Offset: 0x000052DC
		private static void OnColumnCodeCompletion(object target, ExecutedRoutedEventArgs args)
		{
			args.Handled = true;
			CodeTextEditor codeTextEditor = target as CodeTextEditor;
			if (codeTextEditor == null)
			{
				return;
			}
			string wordBeforeCaret = codeTextEditor.GetWordBeforeCaret();
			if (wordBeforeCaret.EndsWith("."))
			{
				int nextCaretPosition = TextUtilities.GetNextCaretPosition(codeTextEditor.Document, codeTextEditor.CaretOffset - 1, LogicalDirection.Backward, CaretPositioningMode.WordStart);
				string text = codeTextEditor.Document.GetText(nextCaretPosition, codeTextEditor.CaretOffset - 1 - nextCaretPosition);
				codeTextEditor.ShowColumnCompletionWindow(text);
			}
		}

		// Token: 0x060000AC RID: 172 RVA: 0x00007143 File Offset: 0x00005343
		private void OnRefreshScreen(object target, ExecutedRoutedEventArgs args)
		{
			EventAggregatorManager.Global.GetEvent<RefreshScreen>().Publish(Application.Current.MainWindow.Tag as PackageKey);
		}

		// Token: 0x060000AD RID: 173 RVA: 0x00007168 File Offset: 0x00005368
		private void OnRefreshDiffResult(object target, ExecutedRoutedEventArgs args)
		{
			EventAggregatorManager.Global.GetEvent<RefreshDiffResult>().Publish(Application.Current.MainWindow.Tag as PackageKey);
		}

		// Token: 0x060000AE RID: 174 RVA: 0x000071A8 File Offset: 0x000053A8
		private void OnSaveDiffStardandResult(string modelName)
		{
			IEnumerable<AddPointModel> enumerable = ResourceController.GetInstance().GetProgramInfo(base.ProgramKey).AddPoints.Where<AddPointModel>((AddPointModel adp) => adp.Name == modelName);
			for (int i = 0; i < enumerable.Count<AddPointModel>(); i++)
			{
				AddPointModel addPointModel = enumerable.ElementAt<AddPointModel>(i);
				SegmentObject segmentObject = base.TextArea.Document.SectionProvider.Find(addPointModel.ID);
				int num = segmentObject.Offset;
				int num2 = segmentObject.EndOffset;
				if (addPointModel.IsSelfDefinition && segmentObject.HasChild)
				{
					num = segmentObject.Children[1].Offset;
					num2 = segmentObject.Children[1].EndOffset;
				}
				DocumentLine lineByNumber = base.TextArea.Document.GetLineByNumber(num);
				DocumentLine lineByNumber2 = base.TextArea.Document.GetLineByNumber(num2);
				base.TextArea.Selection = Selection.Create(base.TextArea, lineByNumber.Offset, lineByNumber2.EndOffset);
				base.TextArea.Selection.ReplaceSelectionWithText(addPointModel.Content.ToString());
			}
		}

		// Token: 0x060000AF RID: 175 RVA: 0x000072D4 File Offset: 0x000054D4
		private void OnProgramErrorCheck(PackageKey key)
		{
			new ErrorCheckWindow(this)
			{
				Owner = Application.Current.MainWindow
			}.ShowDialog();
		}

		// Token: 0x060000B0 RID: 176 RVA: 0x00007300 File Offset: 0x00005500
		private void OnSaveSqlResult(string sql)
		{
			DocumentLine lineByOffset = base.TextArea.Document.GetLineByOffset(base.TextArea.Caret.Offset);
			string text = "".PadLeft(base.TextArea.Caret.Offset - lineByOffset.Offset, ' ');
			string[] array = sql.Split(new string[] { "\r\n" }, StringSplitOptions.RemoveEmptyEntries);
			for (int i = 0; i < array.Length; i++)
			{
				if (i == 0)
				{
					sql = array[i] + "\r\n";
				}
				else
				{
					sql = sql + text + array[i] + "\r\n";
				}
			}
			base.TextArea.Document.Insert(base.TextArea.Caret.Offset, sql);
		}

		// Token: 0x060000B1 RID: 177 RVA: 0x000073C1 File Offset: 0x000055C1
		private void OnRefreshStardandDiffResult(PackageKey key)
		{
			EventAggregatorManager.Global.GetEvent<RefreshDiffResult>().Publish(key);
		}

		// Token: 0x060000B2 RID: 178 RVA: 0x000073D4 File Offset: 0x000055D4
		private static void CanShowFunctionDetail(object sender, CanExecuteRoutedEventArgs e)
		{
			TextArea textArea = CodeTextEditor.GetTextArea(sender);
			string text = textArea.Selection.GetText();
			if (string.IsNullOrEmpty(text))
			{
				e.CanExecute = false;
			}
			else
			{
				e.CanExecute = true;
			}
			CustomizedRoutedCommand customizedRoutedCommand = e.Command as CustomizedRoutedCommand;
			customizedRoutedCommand.Parameter = text;
			e.Handled = true;
		}

		// Token: 0x060000B3 RID: 179 RVA: 0x00007428 File Offset: 0x00005628
		private static void OnShowFunction(object target, ExecutedRoutedEventArgs args)
		{
			string text = (args.Command as CustomizedRoutedCommand).Parameter as string;
			EventAggregatorManager.Global.GetEvent<RunCommandEvent>().Publish(string.Format("adzp280 {0}", text));
		}

		// Token: 0x060000B4 RID: 180 RVA: 0x000074A4 File Offset: 0x000056A4
		private static void CanFramMark(object sender, CanExecuteRoutedEventArgs e)
		{
			e.CanExecute = false;
			CodeTextEditor codeTextEditor = sender as CodeTextEditor;
			if (codeTextEditor.Mode == ContentType.ADP)
			{
				ReadOnlyCollection<SegmentObject> segments = codeTextEditor.Document.SectionProvider.FindOverlappingSegments(codeTextEditor.CaretOffset);
				DocumentLine lineByOffset = codeTextEditor.Document.GetLineByOffset(codeTextEditor.CaretOffset);
				SegmentObject section = codeTextEditor.Document.SectionProvider.Find(ContentType.SEC, lineByOffset.LineNumber);
				SectionModel sectionModel = null;
				if (section != null)
				{
					sectionModel = ResourceController.GetInstance().GetProgramInfo(codeTextEditor.ProgramKey).Sections.Where<SectionModel>((SectionModel sec) => sec.ID == section.ID).ElementAtOrDefault<SectionModel>(0);
				}
				if (segments.Count > 0)
				{
					IEnumerable<AddPointModel> enumerable = ResourceController.GetInstance().GetProgramInfo(codeTextEditor.ProgramKey).AddPoints.Where<AddPointModel>((AddPointModel ap) => ap.ID == segments.ElementAtOrDefault<SegmentObject>(0).ID);
					if (enumerable.Count<AddPointModel>() > 0)
					{
						if (sectionModel != null && sectionModel.IsCustomized)
						{
							e.CanExecute = false;
							return;
						}
						if (enumerable.ElementAtOrDefault<AddPointModel>(0).IsMarkable == YesNo.N)
						{
							e.CanExecute = false;
							return;
						}
						CustomizedRoutedCommand customizedRoutedCommand = e.Command as CustomizedRoutedCommand;
						customizedRoutedCommand.Parameter = enumerable.ElementAtOrDefault<AddPointModel>(0).Name;
						e.CanExecute = true;
					}
				}
			}
		}

		// Token: 0x060000B5 RID: 181 RVA: 0x00007624 File Offset: 0x00005824
		private static void OnFramMark(object target, ExecutedRoutedEventArgs args)
		{
			CodeTextEditor codeTextEditor = target as CodeTextEditor;
			string name = (args.Command as CustomizedRoutedCommand).Parameter as string;
			AddPointModel addPointModel = ResourceController.GetInstance().GetProgramInfo(codeTextEditor.ProgramKey).AddPoints.Where<AddPointModel>((AddPointModel ap) => ap.Name == name).ElementAtOrDefault<AddPointModel>(0);
			if (addPointModel.IsMarkHard)
			{
				addPointModel.IsMarkHard = false;
			}
			else
			{
				addPointModel.IsMarkHard = true;
			}
			if (addPointModel.IsMarkHard)
			{
				codeTextEditor.AddMarkedArea(codeTextEditor.CaretOffset);
				return;
			}
			codeTextEditor.RemoveMarkedArea(codeTextEditor.CaretOffset);
		}

		// Token: 0x060000B6 RID: 182 RVA: 0x000076F0 File Offset: 0x000058F0
		public static void CanFunctionModify(object sender, CanExecuteRoutedEventArgs e)
		{
			e.Handled = true;
			e.CanExecute = false;
			CodeTextEditor codeTextEditor = sender as CodeTextEditor;
			if (codeTextEditor == null)
			{
				return;
			}
			if (codeTextEditor.Mode == ContentType.SEC)
			{
				e.CanExecute = false;
				return;
			}
			int lineNumber = codeTextEditor.Document.GetLineByOffset(codeTextEditor.CaretOffset).LineNumber;
			EditObject g = codeTextEditor.Document.SectionProvider.FindContains(lineNumber) as EditObject;
			if (null == g)
			{
				return;
			}
			AddPointModel addPointModel = ResourceController.GetInstance().GetProgramInfo(codeTextEditor.ProgramKey).AddPoints.Where<AddPointModel>((AddPointModel m) => m.ID == g.ID && (m.Status & Status.DELETE) == Status.NULL).ElementAtOrDefault<AddPointModel>(0);
			if (addPointModel != null && addPointModel.IsSelfDefinition && addPointModel.IsEditable)
			{
				e.CanExecute = true;
			}
		}

		// Token: 0x060000B7 RID: 183 RVA: 0x000077E8 File Offset: 0x000059E8
		public static void ExecutedRename(object sender, ExecutedRoutedEventArgs e)
		{
			e.Handled = true;
			CodeTextEditor codeTextEditor = sender as CodeTextEditor;
			int lineNumber = codeTextEditor.Document.GetLineByOffset(codeTextEditor.CaretOffset).LineNumber;
			EditObject g = codeTextEditor.Document.SectionProvider.FindContains(lineNumber) as EditObject;
			AddPointModel addPointModel = ResourceController.GetInstance().GetProgramInfo(codeTextEditor.ProgramKey).AddPoints.Where<AddPointModel>((AddPointModel a) => a.ID == g.ID && (a.Status & Status.DELETE) == Status.NULL).ElementAtOrDefault<AddPointModel>(0);
			addPointModel.Content = codeTextEditor.Document.SectionProvider.GetText(g.ID);
			FunctionInfoWindow functionInfoWindow = new FunctionInfoWindow();
			functionInfoWindow.Show(addPointModel.Clone());
		}

		// Token: 0x060000B8 RID: 184 RVA: 0x000078D0 File Offset: 0x00005AD0
		public static void ExecutedDelete(object sender, ExecutedRoutedEventArgs e)
		{
			e.Handled = true;
			CodeTextEditor codeTextEditor = sender as CodeTextEditor;
			int lineNumber = codeTextEditor.Document.GetLineByOffset(codeTextEditor.CaretOffset).LineNumber;
			EditObject g = codeTextEditor.Document.SectionProvider.Find(lineNumber) as EditObject;
			if (g == null)
			{
				throw new Exception("No Deletable");
			}
			AddPointModel addPointModel = ResourceController.GetInstance().GetProgramInfo(codeTextEditor.ProgramKey).AddPoints.Where<AddPointModel>((AddPointModel a) => a.ID == g.ID && (a.Status & Status.DELETE) == Status.NULL).ElementAtOrDefault<AddPointModel>(0);
			EventController.GetInstance().GetEvent<DeleteFunctionEvent>().Publish(addPointModel.Name);
			EventAggregatorManager.Get(codeTextEditor.ProgramKey).GetEvent<CodeChangedEvent>().Publish(codeTextEditor.ProgramKey);
		}

		// Token: 0x060000B9 RID: 185 RVA: 0x0000799C File Offset: 0x00005B9C
		public static void ExecutedGoToLine(object sender, ExecutedRoutedEventArgs e)
		{
			e.Handled = true;
			CodeTextEditor codeTextEditor = sender as CodeTextEditor;
			GoToLineWindow goToLineWindow = new GoToLineWindow(codeTextEditor);
			if (goToLineWindow.ShowDialog() == true)
			{
				DocumentLine lineByNumber = codeTextEditor.Document.GetLineByNumber(int.Parse(goToLineWindow.LineOnCaret));
				codeTextEditor.CaretOffset = lineByNumber.Offset;
				codeTextEditor.ScrollTo(lineByNumber.LineNumber);
			}
			goToLineWindow.Dispose();
		}

		// Token: 0x060000BA RID: 186 RVA: 0x00007A10 File Offset: 0x00005C10
		public static void ExecutedGoToDiffLine(object sender, ExecutedRoutedEventArgs e)
		{
			e.Handled = true;
			CodeTextEditor codeTextEditor = sender as CodeTextEditor;
			GoToDiffLineWindow goToDiffLineWindow = new GoToDiffLineWindow(codeTextEditor);
			if (goToDiffLineWindow.ShowDialog() == true)
			{
				for (int i = 1; i <= codeTextEditor.LineCount; i++)
				{
					DocumentLine lineByNumber = codeTextEditor.Document.GetLineByNumber(i);
					if (lineByNumber.RealLineNumber == int.Parse(goToDiffLineWindow.RealLineOnCaret))
					{
						codeTextEditor.CaretOffset = lineByNumber.Offset;
						codeTextEditor.ScrollTo(lineByNumber.LineNumber);
						break;
					}
				}
			}
			goToDiffLineWindow.Dispose();
		}

		// Token: 0x060000BB RID: 187 RVA: 0x00007AA0 File Offset: 0x00005CA0
		private static void CanInsertCodeSample(object sender, CanExecuteRoutedEventArgs e)
		{
			CodeTextEditor codeTextEditor = sender as CodeTextEditor;
			bool flag = CodeTextEditor.IsEditable(codeTextEditor);
			if (flag)
			{
				ReadOnlyCollection<SegmentObject> readOnlyCollection = codeTextEditor.Document.SectionProvider.FindOverlappingSegments(codeTextEditor.CaretOffset);
				if (readOnlyCollection.Count > 0)
				{
					CustomizedRoutedCommand customizedRoutedCommand = e.Command as CustomizedRoutedCommand;
					int num = 0;
					bool flag2 = int.TryParse(customizedRoutedCommand.Parameter as string, out num);
					if (flag2)
					{
						if (num == 0)
						{
							flag = false;
						}
					}
					else
					{
						customizedRoutedCommand.Parameter = ((e.Parameter == null) ? string.Empty : e.Parameter.ToString());
					}
				}
				else
				{
					flag = false;
				}
			}
			e.CanExecute = flag;
		}

		// Token: 0x060000BC RID: 188 RVA: 0x00007B54 File Offset: 0x00005D54
		private static void ExecutedInsertCodeSample(object sender, ExecutedRoutedEventArgs e)
		{
			e.Handled = true;
			string id = (e.Command as CustomizedRoutedCommand).Parameter as string;
			CodeSamepleModel codeSamepleModel = ResourceController.GetInstance().CodeSamples.Where<CodeSamepleModel>((CodeSamepleModel cs) => cs.ID == id).ElementAtOrDefault<CodeSamepleModel>(0);
			if (codeSamepleModel == null)
			{
				return;
			}
			CodeTextEditor codeTextEditor = sender as CodeTextEditor;
			DocumentLine lineByOffset = codeTextEditor.Document.GetLineByOffset(codeTextEditor.CaretOffset);
			int num = codeTextEditor.CaretOffset - lineByOffset.Offset;
			string text = new string(' ', num);
			codeTextEditor.Document.UndoStack.Pause();
			if (codeTextEditor.SelectionLength > 0)
			{
				codeTextEditor.Document.Remove(codeTextEditor.SelectionStart, codeTextEditor.SelectionLength);
			}
			codeTextEditor.Document.UndoStack.Resume();
			Regex regex = new Regex("^.*$", RegexOptions.Multiline);
			Match match = regex.Match(codeSamepleModel.Content);
			int num2 = ((match.Index + match.Length + 1 < codeSamepleModel.Content.Length) ? (match.Index + match.Length + 1) : codeSamepleModel.Content.Length);
			string text2 = regex.Replace(codeSamepleModel.Content, text + "$&", int.MaxValue, num2);
			codeTextEditor.Document.Insert(codeTextEditor.CaretOffset, text2);
		}

		// Token: 0x060000BD RID: 189 RVA: 0x00007CE4 File Offset: 0x00005EE4
		private static void CanCiteOrNot(object sender, CanExecuteRoutedEventArgs e)
		{
			CodeTextEditor codeTextEditor = sender as CodeTextEditor;
			if (codeTextEditor.Mode == ContentType.SEC)
			{
				e.CanExecute = false;
				return;
			}
			AddPointModel addPointModel = null;
			if (!SettingManager.Get().GetTzpManger(codeTextEditor.ProgramKey).IsStandardProgram)
			{
				IEnumerable<ISegment> enumerable = codeTextEditor.TextArea.Selection.Segments.Cast<ISegment>();
				if (enumerable.Count<ISegment>() == 0)
				{
					SegmentObject editableGroup = codeTextEditor.Document.SectionProvider.FindOverlappingSegments(codeTextEditor.CaretOffset).ElementAtOrDefault<SegmentObject>(0);
					if (editableGroup != null && editableGroup is EditObject)
					{
						addPointModel = ResourceController.GetInstance().GetProgramInfo(Application.Current.MainWindow.Tag as PackageKey).AddPoints.Where<AddPointModel>((AddPointModel ap) => ap.ID == editableGroup.ID && (ap.Status & Status.DELETE) != Status.DELETE).ElementAtOrDefault<AddPointModel>(0);
						if (!CitedAddPointHelper.HasCitedContent(codeTextEditor.ProgramKey, addPointModel.Mapping))
						{
							addPointModel = null;
						}
					}
				}
			}
			e.CanExecute = addPointModel != null;
			CustomizedRoutedCommand customizedRoutedCommand = e.Command as CustomizedRoutedCommand;
			if (customizedRoutedCommand != null)
			{
				customizedRoutedCommand.Parameter = addPointModel;
			}
		}

		// Token: 0x060000BE RID: 190 RVA: 0x00007E0C File Offset: 0x0000600C
		private static void OnCiteOrNot(object target, ExecutedRoutedEventArgs args)
		{
			CodeTextEditor codeTextEditor = target as CodeTextEditor;
			AddPointModel addPointModel = (args.Command as CustomizedRoutedCommand).Parameter as AddPointModel;
			if (YesNo.N == addPointModel.CiteSetting)
			{
				addPointModel.CiteSetting = YesNo.Y;
			}
			else if (YesNo.Y == addPointModel.CiteSetting)
			{
				addPointModel.CiteSetting = YesNo.N;
				return;
			}
			codeTextEditor.Document.UndoStack.Pause();
			SegmentObject segmentObject = codeTextEditor.Document.SectionProvider.Find(addPointModel.ID);
			TextSegment textSegment;
			if (segmentObject.Children != null)
			{
				textSegment = new TextSegment
				{
					StartOffset = codeTextEditor.Document.GetOffset(segmentObject.Children[1].Offset, 0),
					EndOffset = codeTextEditor.Document.GetLineByNumber(segmentObject.Children[1].EndOffset).EndOffset
				};
			}
			else
			{
				textSegment = new TextSegment
				{
					StartOffset = codeTextEditor.Document.GetOffset(segmentObject.Offset, 0),
					EndOffset = codeTextEditor.Document.GetLineByNumber(segmentObject.EndOffset).EndOffset
				};
			}
			string text = string.Empty;
			text = CitedAddPointHelper.GetCitedContent(addPointModel.ProgramKey, addPointModel.Mapping);
			if (SettingManager.Get().GetTzpManger(codeTextEditor.ProgramKey).ProgType.Equals("S") && !SettingManager.Get().GetTzpManger(codeTextEditor.ProgramKey).IsStandardProgram)
			{
				string programName = SettingManager.Get().GetTzpManger(codeTextEditor.ProgramKey).ProgramName;
				string stdProgramName = SettingManager.Get().GetTzpManger(codeTextEditor.ProgramKey).StdProgramName;
				text = text.Replace(stdProgramName, programName);
			}
			addPointModel.ContentData = text;
			if (text != null)
			{
				codeTextEditor.Document.Replace(textSegment, addPointModel.ContentData);
			}
			codeTextEditor.Document.UndoStack.Resume();
			codeTextEditor.TextArea.TextView.InvalidateMeasure();
			codeTextEditor.TextArea.TextView.Redraw(textSegment, DispatcherPriority.Normal);
		}

		// Token: 0x060000BF RID: 191 RVA: 0x00008000 File Offset: 0x00006200
		private static void CanCopyAllColumnsToRecord(object sender, CanExecuteRoutedEventArgs e)
		{
			e.Handled = true;
			CodeTextEditor codeTextEditor = sender as CodeTextEditor;
			ISegment surroundingSegment = codeTextEditor.TextArea.Selection.SurroundingSegment;
			if (surroundingSegment != null)
			{
				string text = codeTextEditor.Document.GetText(surroundingSegment.Offset, surroundingSegment.EndOffset - surroundingSegment.Offset);
				if (text.Length > 0 && TableColumnHelper.GetColFields(text) != null && TableColumnHelper.GetColFields(text).Count<XElement>() > 0)
				{
					e.CanExecute = true;
					return;
				}
			}
			e.CanExecute = false;
		}

		// Token: 0x060000C0 RID: 192 RVA: 0x0000807C File Offset: 0x0000627C
		private static void OnCopyAllColumnsToRecord(object target, ExecutedRoutedEventArgs args)
		{
			CodeTextEditor codeTextEditor = target as CodeTextEditor;
			if (codeTextEditor == null)
			{
				return;
			}
			ISegment surroundingSegment = codeTextEditor.TextArea.Selection.SurroundingSegment;
			string text = codeTextEditor.Document.GetText(surroundingSegment.Offset, surroundingSegment.EndOffset - surroundingSegment.Offset);
			StringBuilder stringBuilder = new StringBuilder();
			stringBuilder.Append(string.Concat(new string[]
			{
				"DEFINE l_",
				text.Split(new char[] { '_' })[0],
				" RECORD  #",
				TableColumnHelper.GetTableDesc(text),
				"\r\n"
			}));
			foreach (XElement xelement in TableColumnHelper.GetColFields(text))
			{
				XAttribute xattribute = xelement.Attribute("name");
				if (xattribute != null)
				{
					string columnText = TableColumnHelper.GetColumnText(text, xattribute.Value);
					stringBuilder.Append(string.Concat(new string[]
					{
						"       ",
						xelement.Attribute("name").Value,
						" LIKE ",
						text,
						".",
						xattribute.Value,
						", #",
						columnText,
						"\r\n"
					}));
				}
			}
			stringBuilder.Append("END RECORD");
			if (stringBuilder.Length > 0)
			{
				Clipboard.Clear();
				Clipboard.SetDataObject(stringBuilder.ToString());
			}
		}

		// Token: 0x060000C1 RID: 193 RVA: 0x00008228 File Offset: 0x00006428
		private static void CanCopyColumnsName(object sender, CanExecuteRoutedEventArgs e)
		{
			e.Handled = true;
			CodeTextEditor codeTextEditor = sender as CodeTextEditor;
			ISegment surroundingSegment = codeTextEditor.TextArea.Selection.SurroundingSegment;
			if (surroundingSegment != null)
			{
				string text = codeTextEditor.Document.GetText(surroundingSegment.Offset, surroundingSegment.EndOffset - surroundingSegment.Offset);
				if (text.Length > 0 && TableColumnHelper.GetColFields(text) != null && TableColumnHelper.GetColFields(text).Count<XElement>() > 0)
				{
					e.CanExecute = true;
					return;
				}
			}
			e.CanExecute = false;
		}

		// Token: 0x060000C2 RID: 194 RVA: 0x000082A4 File Offset: 0x000064A4
		private static void OnCopyColumnsName(object target, ExecutedRoutedEventArgs args)
		{
			CodeTextEditor codeTextEditor = target as CodeTextEditor;
			if (codeTextEditor == null)
			{
				return;
			}
			ISegment surroundingSegment = codeTextEditor.TextArea.Selection.SurroundingSegment;
			string text = codeTextEditor.Document.GetText(surroundingSegment.Offset, surroundingSegment.EndOffset - surroundingSegment.Offset);
			StringBuilder stringBuilder = new StringBuilder();
			foreach (XElement xelement in TableColumnHelper.GetColFields(text))
			{
				XAttribute xattribute = xelement.Attribute("name");
				if (xattribute != null)
				{
					if (stringBuilder.Length > 0)
					{
						stringBuilder.Append(",");
					}
					stringBuilder.Append(xelement.Attribute("name").Value);
				}
			}
			if (stringBuilder.Length > 0)
			{
				Clipboard.Clear();
				Clipboard.SetDataObject(stringBuilder.ToString());
			}
		}

		// Token: 0x060000C3 RID: 195 RVA: 0x00008394 File Offset: 0x00006594
		private static void CanReturnToStandard(object sender, CanExecuteRoutedEventArgs e)
		{
			CodeTextEditor codeTextEditor = sender as CodeTextEditor;
			if (codeTextEditor.Mode == ContentType.SEC)
			{
				e.CanExecute = false;
			}
			else
			{
				int lineNumber = codeTextEditor.Document.GetLineByOffset(codeTextEditor.CaretOffset).LineNumber;
				SegmentObject segmentObject = codeTextEditor.Document.SectionProvider.Find(lineNumber);
				if (null != segmentObject && ResourceController.GetInstance().GetProgramInfo(codeTextEditor.ProgramKey).StdToCus.Equals("Y"))
				{
					e.CanExecute = true;
					return;
				}
			}
			e.CanExecute = false;
		}

		// Token: 0x060000C4 RID: 196 RVA: 0x0000841C File Offset: 0x0000661C
		private static void OnReturnToStandard(object target, ExecutedRoutedEventArgs args)
		{
			CodeTextEditor codeTextEditor = target as CodeTextEditor;
			if (codeTextEditor == null)
			{
				return;
			}
			string text = string.Format("adzp064 {0} {1}", codeTextEditor.ProgramKey.Program, SettingManager.Get().GetTzpManger(codeTextEditor.ProgramKey).ProgType);
			ConnectionManager.RunProgram(text);
		}

		// Token: 0x060000C5 RID: 197 RVA: 0x00008488 File Offset: 0x00006688
		private static void CanDiffSingleContent(object sender, CanExecuteRoutedEventArgs e)
		{
			e.Handled = true;
			CodeTextEditor codeTextEditor = sender as CodeTextEditor;
			if (SettingManager.Get().GetTzpManger(codeTextEditor.ProgramKey).IsDiff)
			{
				DocumentLine lineByOffset = codeTextEditor.Document.GetLineByOffset(codeTextEditor.CaretOffset);
				SegmentObject segObj = codeTextEditor.Document.SectionProvider.Find(ContentType.ADP, lineByOffset.LineNumber);
				if (segObj != null)
				{
					AddPointModel addPointModel = ResourceController.GetInstance().GetProgramInfo(codeTextEditor.ProgramKey).AddPoints.Where<AddPointModel>((AddPointModel ap) => ap.ID == segObj.ID).FirstOrDefault<AddPointModel>();
					if (addPointModel != null)
					{
						string text = ResourceController.GetInstance().GetProgramInfo(codeTextEditor.ProgramKey).FindDiffSourceContetByName(addPointModel.Name);
						if (text != null)
						{
							CustomizedRoutedCommand customizedRoutedCommand = e.Command as CustomizedRoutedCommand;
							customizedRoutedCommand.Parameter = new CodeTextEditor.DiffContentStructure(text, addPointModel.ToString());
							e.CanExecute = true;
							return;
						}
					}
				}
			}
			e.CanExecute = false;
		}

		// Token: 0x060000C6 RID: 198 RVA: 0x0000858C File Offset: 0x0000678C
		private static void OnDiffSingleContent(object target, ExecutedRoutedEventArgs args)
		{
			CodeTextEditor.DiffContentStructure diffContentStructure = (args.Command as CustomizedRoutedCommand).Parameter as CodeTextEditor.DiffContentStructure;
			StandardVersionComparisonWindow.This.Show(diffContentStructure.SourceContent, diffContentStructure.TargetContent);
		}

		// Token: 0x060000C7 RID: 199 RVA: 0x000085E8 File Offset: 0x000067E8
		private static void CanDiffBaseOnStandard(object sender, CanExecuteRoutedEventArgs e)
		{
			e.Handled = true;
			CodeTextEditor codeTextEditor = sender as CodeTextEditor;
			if (SettingManager.Get().GetTzpManger(codeTextEditor.ProgramKey).IsDiff)
			{
				DocumentLine lineByOffset = codeTextEditor.Document.GetLineByOffset(codeTextEditor.CaretOffset);
				SegmentObject segObj = codeTextEditor.Document.SectionProvider.Find(ContentType.ADP, lineByOffset.LineNumber);
				if (segObj != null && segObj.IsEditable)
				{
					AddPointModel addPointModel = ResourceController.GetInstance().GetProgramInfo(codeTextEditor.ProgramKey).AddPoints.Where<AddPointModel>((AddPointModel ap) => ap.ID == segObj.ID).FirstOrDefault<AddPointModel>();
					if (addPointModel != null)
					{
						string text = string.Empty;
						if (addPointModel.DiffBaseOnStandardModify == "N")
						{
							text = ResourceController.GetInstance().GetProgramInfo(codeTextEditor.ProgramKey).FindDiffSourceContetByName(addPointModel.Name);
							if (addPointModel.IsSelfDefinition && !string.IsNullOrEmpty(text))
							{
								text = new AddPointModel(codeTextEditor.ProgramKey)
								{
									Content = text
								}.Content;
							}
						}
						else
						{
							text = addPointModel.Content;
							if (addPointModel.IsSelfDefinition && !string.IsNullOrEmpty(addPointModel.Content))
							{
								text = new AddPointModel(codeTextEditor.ProgramKey)
								{
									Content = addPointModel.Content
								}.Content;
							}
						}
						if (text != null)
						{
							CustomizedRoutedCommand customizedRoutedCommand = e.Command as CustomizedRoutedCommand;
							customizedRoutedCommand.Parameter = new CodeTextEditor.DiffBaseOnStandardStructure(addPointModel.OriContent, text, addPointModel);
							e.CanExecute = true;
							return;
						}
					}
				}
			}
			e.CanExecute = false;
		}

		// Token: 0x060000C8 RID: 200 RVA: 0x0000878C File Offset: 0x0000698C
		private static void OnDiffBaseOnStandard(object target, ExecutedRoutedEventArgs args)
		{
			CodeTextEditor codeTextEditor = target as CodeTextEditor;
			CodeTextEditor.DiffBaseOnStandardStructure diffBaseOnStandardStructure = (args.Command as CustomizedRoutedCommand).Parameter as CodeTextEditor.DiffBaseOnStandardStructure;
			DiffBaseOnStandardWindow diffBaseOnStandardWindow = new DiffBaseOnStandardWindow();
			diffBaseOnStandardWindow.Show(diffBaseOnStandardStructure.SourceContent, diffBaseOnStandardStructure.TargetContent, codeTextEditor.ProgramKey, diffBaseOnStandardStructure.APT);
			diffBaseOnStandardWindow.ShowDialog();
		}

		// Token: 0x060000C9 RID: 201 RVA: 0x000087E4 File Offset: 0x000069E4
		private static void CanFullTextKeywordUpper(object sender, CanExecuteRoutedEventArgs e)
		{
			e.Handled = true;
			e.CanExecute = true;
			CodeTextEditor codeTextEditor = sender as CodeTextEditor;
			if (codeTextEditor.ProgramKey == null)
			{
				e.CanExecute = false;
			}
		}

		// Token: 0x060000CA RID: 202 RVA: 0x00008848 File Offset: 0x00006A48
		private static void OnFullTextKeywordUpper(object target, ExecutedRoutedEventArgs args)
		{
			CodeTextEditor.<>c__DisplayClass39 CS$<>8__locals1 = new CodeTextEditor.<>c__DisplayClass39();
			CodeTextEditor codeTextEditor = target as CodeTextEditor;
			EventAggregatorManager.Global.GetEvent<SaveAddPointEvent>().Publish(codeTextEditor.ProgramKey);
			List<AddPointModel> list = ResourceController.GetInstance().GetProgramInfo(codeTextEditor.ProgramKey).AddPoints.Where<AddPointModel>((AddPointModel ap) => (ap.Status & Status.DELETE) == Status.NULL && ap.IsLoaded).ToList<AddPointModel>();
			StringBuilder stringBuilder = new StringBuilder();
			CS$<>8__locals1.word = string.Empty;
			bool flag = false;
			foreach (AddPointModel addPointModel in list)
			{
				stringBuilder.Clear();
				string text = codeTextEditor.TextArea.Document.SectionProvider.GetText(addPointModel.ID);
				string text2 = string.Empty;
				string text3 = string.Empty;
				string[] array;
				if (addPointModel.Type == DefinitionType.FUNCTION || addPointModel.Type == DefinitionType.DIALOG || addPointModel.Type == DefinitionType.REPORT)
				{
					text2 = codeTextEditor.TextArea.Document.SectionProvider.GetText(addPointModel.ID).Replace(addPointModel.Description + "\r\n", "");
					text2 = CodeTextEditor.scopeRegex.Replace(text2, "", 1);
					switch (addPointModel.Type)
					{
					case DefinitionType.FUNCTION:
						text2 = CodeTextEditor.funcReg.Replace(text2, "", 1);
						text2 = CodeTextEditor.endFuncReg.Replace(text2, "", 1);
						break;
					case DefinitionType.DIALOG:
						text2 = CodeTextEditor.dialogReg.Replace(text2, "", 1);
						text2 = CodeTextEditor.enddialogReg.Replace(text2, "", 1);
						break;
					case DefinitionType.REPORT:
						text2 = CodeTextEditor.reportReg.Replace(text2, "", 1);
						text2 = CodeTextEditor.endreportReg.Replace(text2, "", 1);
						break;
					}
					array = text2.Split(new string[] { "\n" }, StringSplitOptions.RemoveEmptyEntries);
				}
				else
				{
					array = text.Split(new string[] { "\n" }, StringSplitOptions.RemoveEmptyEntries);
				}
				for (int i = 0; i < array.Length; i++)
				{
					text3 = array[i];
					for (int j = 1; j <= text3.Length; j++)
					{
						char c = Convert.ToChar(text3.Substring(j - 1, 1));
						if (!char.IsWhiteSpace(c))
						{
							CodeTextEditor.<>c__DisplayClass39 CS$<>8__locals2 = CS$<>8__locals1;
							CS$<>8__locals2.word += c;
							if (c == '#')
							{
								stringBuilder.Append(CS$<>8__locals1.word);
								stringBuilder.Append(text3.Substring(j, text3.Length - j));
								CS$<>8__locals1.word = string.Empty;
								break;
							}
						}
						else
						{
							if (ResourceController.GetInstance().FglKeywords.Any<string>((string s) => string.Equals(s, CS$<>8__locals1.word, StringComparison.InvariantCultureIgnoreCase)))
							{
								stringBuilder.Append(CS$<>8__locals1.word.ToUpper());
							}
							else
							{
								stringBuilder.Append(CS$<>8__locals1.word);
							}
							stringBuilder.Append(c);
							CS$<>8__locals1.word = string.Empty;
						}
					}
					if (!string.IsNullOrEmpty(CS$<>8__locals1.word))
					{
						stringBuilder.Append(CS$<>8__locals1.word);
						CS$<>8__locals1.word = string.Empty;
					}
					if (i < array.Length - 1)
					{
						stringBuilder.Append("\r\n");
					}
				}
				if (addPointModel.Type == DefinitionType.FUNCTION || addPointModel.Type == DefinitionType.DIALOG || addPointModel.Type == DefinitionType.REPORT)
				{
					string text4 = text.Replace(text2, stringBuilder.ToString());
					stringBuilder.Clear();
					stringBuilder.Append(text4);
				}
				if (text.Replace("\r", "").Replace("\n", "") != stringBuilder.ToString().Replace("\r", "").Replace("\n", ""))
				{
					addPointModel.Content = stringBuilder.ToString();
					flag = true;
				}
			}
			if (flag)
			{
				EventAggregatorManager.Global.GetEvent<FullTextRefreshScreen>().Publish(Application.Current.MainWindow.Tag as PackageKey);
			}
		}

		// Token: 0x060000CB RID: 203 RVA: 0x00008CA0 File Offset: 0x00006EA0
		private static void CanGenerateSqlTemplate(object sender, CanExecuteRoutedEventArgs e)
		{
			e.Handled = true;
			CodeTextEditor codeTextEditor = sender as CodeTextEditor;
			e.CanExecute = CodeTextEditor.IsEditable(codeTextEditor);
		}

		// Token: 0x060000CC RID: 204 RVA: 0x00008CC8 File Offset: 0x00006EC8
		private static void OnGenerateSqlTemplate(object target, ExecutedRoutedEventArgs args)
		{
			GenerateSqlTemplateWindow generateSqlTemplateWindow = new GenerateSqlTemplateWindow();
			generateSqlTemplateWindow.ShowDialog();
		}

		// Token: 0x060000CD RID: 205 RVA: 0x00008CE4 File Offset: 0x00006EE4
		private static bool IsEditable(CodeTextEditor editor)
		{
			bool flag = true;
			Selection selection = editor.TextArea.Selection;
			if (!selection.IsEmpty)
			{
				using (IEnumerator<SelectionSegment> enumerator = selection.Segments.GetEnumerator())
				{
					while (enumerator.MoveNext())
					{
						SelectionSegment selectionSegment = enumerator.Current;
						int num = selectionSegment.StartOffset;
						while (num <= selectionSegment.EndOffset && flag)
						{
							if (!editor.Document.SectionProvider.CanInsert(num))
							{
								flag = false;
								break;
							}
							num++;
						}
					}
					return flag;
				}
			}
			flag = editor.Document.SectionProvider.CanInsert(editor.CaretOffset);
			return flag;
		}

		// Token: 0x17000015 RID: 21
		// (get) Token: 0x060000CE RID: 206 RVA: 0x00008D90 File Offset: 0x00006F90
		// (set) Token: 0x060000CF RID: 207 RVA: 0x00008D98 File Offset: 0x00006F98
		public DesignerCompletionWindow CompletionWindow
		{
			get
			{
				return this._completionWindow;
			}
			set
			{
				this._completionWindow = value;
			}
		}

		// Token: 0x17000016 RID: 22
		// (get) Token: 0x060000D0 RID: 208 RVA: 0x00008DA1 File Offset: 0x00006FA1
		// (set) Token: 0x060000D1 RID: 209 RVA: 0x00008DA9 File Offset: 0x00006FA9
		private DesignerInsightWindow InsightWindow
		{
			get
			{
				return this._insightWindow;
			}
			set
			{
				this._insightWindow = value;
			}
		}

		// Token: 0x060000D2 RID: 210 RVA: 0x00008DB4 File Offset: 0x00006FB4
		private void InitializeCodeCompletion()
		{
			IList<ICompletionData> completionData = this.list.CompletionData;
			foreach (IIntellisenseModel intellisenseModel in ResourceController.GetInstance().KeywordSuggestion)
			{
				CodeCompletionData codeCompletionData = CodeCompletionData.Create(intellisenseModel);
				completionData.Add(codeCompletionData);
			}
			this._codeCompletion = new GeneroCodeCompletionBinding(this, completionData);
		}

		// Token: 0x060000D3 RID: 211 RVA: 0x00008E5C File Offset: 0x0000705C
		private void CreateContextMenu()
		{
			AddPointModel addPointModel = null;
			int lineNumber = this.Document.GetLineByOffset(this.CaretOffset).LineNumber;
			SegmentObject segment = this.Document.SectionProvider.Find(lineNumber);
			if (segment != null)
			{
				addPointModel = ResourceController.GetInstance().GetProgramInfo(this.ProgramKey).AddPoints.Where<AddPointModel>((AddPointModel ap) => ap.ID == segment.ID && (ap.Status & Status.DELETE) == Status.NULL).ElementAtOrDefault<AddPointModel>(0);
			}
			this.SetContextMenu();
			this.SetContextMenuByModel(addPointModel);
		}

		// Token: 0x060000D4 RID: 212 RVA: 0x00008EF4 File Offset: 0x000070F4
		private void SetContextMenu()
		{
			if (base.ContextMenu != null && base.ContextMenu.Items.Count > 0)
			{
				return;
			}
			base.ContextMenu = new ContextMenu();
			base.ContextMenu.Items.Add(new MenuItem
			{
				Header = (Application.Current.FindResource("CE_Cut") as string),
				Command = ApplicationCommands.Cut
			});
			base.ContextMenu.Items.Add(new MenuItem
			{
				Header = (Application.Current.FindResource("CE_Copy") as string),
				Command = ApplicationCommands.Copy
			});
			base.ContextMenu.Items.Add(new MenuItem
			{
				Header = (Application.Current.FindResource("CE_Paste") as string),
				Command = ApplicationCommands.Paste
			});
			base.ContextMenu.Items.Add(new Separator());
			base.ContextMenu.Items.Add(new MenuItem
			{
				Header = (Application.Current.FindResource("CE_ModifyFunction") as string),
				Command = CodeEditCommands.Rename
			});
			base.ContextMenu.Items.Add(new MenuItem
			{
				Header = (Application.Current.FindResource("CE_DeleteFunction") as string),
				Command = CodeEditCommands.DeleteFunction
			});
			base.ContextMenu.Items.Add(new MenuItem
			{
				Header = (Application.Current.FindResource("CE_Block") as string),
				Command = CodeEditCommands.BlockComment
			});
			base.ContextMenu.Items.Add(new MenuItem
			{
				Header = (Application.Current.FindResource("CE_UnBlock") as string),
				Command = CodeEditCommands.UnblockComment
			});
			base.ContextMenu.Items.Add(new MenuItem
			{
				Header = (Application.Current.FindResource("CE_Refresh") as string),
				Command = CodeEditCommands.RefreshScreen
			});
			base.ContextMenu.Items.Add(new Separator());
			base.ContextMenu.Items.Add(new MenuItem
			{
				Header = (Application.Current.FindResource("CE_KeywordConvert") as string),
				Command = CodeEditCommands.KeywordUpperCase
			});
			base.ContextMenu.Items.Add(new MenuItem
			{
				Header = (Application.Current.FindResource("CE_CodeCompletion") as string),
				Command = CodeEditCommands.CodeCompletion
			});
			base.ContextMenu.Items.Add(new MenuItem
			{
				Header = (Application.Current.FindResource("CE_GenerateSqlTemplate") as string),
				Command = CodeEditCommands.GenerateSqlTemplate
			});
			base.ContextMenu.Items.Add(new MenuItem
			{
				Header = (Application.Current.FindResource("CE_ShowFunction") as string),
				Command = CodeEditCommands.ShowFunction
			});
			MenuItem menuItem = new MenuItem();
			menuItem.Header = Application.Current.FindResource("CE_OftenUsed") as string;
			menuItem.Command = CodeEditCommands.InsertCodeSample;
			foreach (CodeSamepleModel codeSamepleModel in ResourceController.GetInstance().CodeSamples)
			{
				if (codeSamepleModel.IsShow)
				{
					menuItem.Items.Add(new MenuItem
					{
						Header = string.Format(Application.Current.FindResource("CE_CodeSample") as string, codeSamepleModel.Description),
						Command = CodeEditCommands.InsertCodeSample,
						CommandParameter = codeSamepleModel.ID
					});
				}
			}
			base.ContextMenu.Items.Add(menuItem);
			base.ContextMenu.Items.Add(new MenuItem
			{
				Header = (Application.Current.FindResource("CE_MarkFrame") as string),
				Command = CodeEditCommands.FramMark,
				IsCheckable = true
			});
			base.ContextMenu.Items.Add(new MenuItem
			{
				Header = (Application.Current.FindResource("CE_CiteOrNot") as string),
				Command = CodeEditCommands.CiteOrNot,
				IsCheckable = true
			});
			base.ContextMenu.Items.Add(new MenuItem
			{
				Header = (Application.Current.FindResource("columnView_CopyAllColumnsToRecord") as string),
				Command = CodeEditCommands.CopyAllColumnsToRecord
			});
			base.ContextMenu.Items.Add(new MenuItem
			{
				Header = (Application.Current.FindResource("columnView_CopyColumnsName") as string),
				Command = CodeEditCommands.CopyColumnsName
			});
			base.ContextMenu.Items.Add(new MenuItem
			{
				Header = (Application.Current.FindResource("CE_ReturnToStandard") as string),
				Command = CodeEditCommands.ReturnToStandard
			});
			if (SettingManager.Get().GetTzpManger(base.ProgramKey).IsDiff)
			{
				if (SettingManager.Get().GetTzpManger(base.ProgramKey).DIFF_TAP != null)
				{
					base.ContextMenu.Items.Add(new MenuItem
					{
						Header = (Application.Current.FindResource("CE_DiffSingleContent") as string),
						Command = CodeEditCommands.DiffSingleContent
					});
				}
				base.ContextMenu.Items.Add(new MenuItem
				{
					Header = (Application.Current.FindResource("CE_DiffBaseOnStandard") as string),
					Command = CodeEditCommands.DiffBaseOnStandard
				});
			}
			base.ContextMenu.Items.Add(new Separator());
			base.ContextMenu.Items.Add(new MenuItem
			{
				Header = "AddPoint Name: --------",
				Command = CodeEditCommands.ShowAddPointName
			});
		}

		// Token: 0x060000D5 RID: 213 RVA: 0x00009584 File Offset: 0x00007784
		private MenuItem FindContextMenuItem(ICommand command)
		{
			if (base.ContextMenu == null)
			{
				return null;
			}
			foreach (object obj in ((IEnumerable)base.ContextMenu.Items))
			{
				MenuItem menuItem = obj as MenuItem;
				if (menuItem != null && menuItem.Command == command)
				{
					return menuItem;
				}
			}
			return null;
		}

		// Token: 0x060000D6 RID: 214 RVA: 0x00009600 File Offset: 0x00007800
		private void SetContextMenuByModel(AddPointModel model)
		{
			MenuItem menuItem = this.FindContextMenuItem(CodeEditCommands.FramMark);
			if (menuItem != null)
			{
				menuItem.IsChecked = model != null && model.IsMarkHard;
			}
			MenuItem menuItem2 = this.FindContextMenuItem(CodeEditCommands.CiteOrNot);
			if (menuItem2 != null)
			{
				menuItem2.IsChecked = model != null && model.CiteSetting == YesNo.Y;
			}
			MenuItem menuItem3 = this.FindContextMenuItem(CodeEditCommands.ShowAddPointName);
			if (menuItem3 != null)
			{
				menuItem3.Header = new TextBlock
				{
					Text = string.Format("{0}{1}", (model != null) ? model.Name : "--------", (model != null) ? (model.IsCustomized ? "(c)" : "(s)") : "")
				};
			}
			MenuItem menuItem4 = this.FindContextMenuItem(CodeEditCommands.DiffBaseOnStandard);
			if (menuItem4 != null)
			{
				new TextBlock();
				if (model != null)
				{
					string text = ResourceController.GetInstance().GetProgramInfo(base.ProgramKey).FindDiffSourceContetByName(model.Name);
					if (string.IsNullOrEmpty(text))
					{
						menuItem4.Header = (Application.Current.FindResource("CE_DiffBaseOnStandard") as string) + " (no standard)";
					}
				}
			}
			MenuItem menuItem5 = this.FindContextMenuItem(CodeEditCommands.FullTextKeywordUpperCase);
			if (menuItem5 != null)
			{
				if (ResourceController.GetInstance().GetProgramInfo(base.ProgramKey).ENV == "s")
				{
					menuItem5.Visibility = Visibility.Visible;
					return;
				}
				menuItem5.Visibility = Visibility.Collapsed;
			}
		}

		// Token: 0x060000D7 RID: 215 RVA: 0x00009758 File Offset: 0x00007958
		public void FocusSearchResult(SearchResult result)
		{
			base.CaretOffset = result.StartOffset;
			base.TextArea.Caret.BringCaretToView();
		}

		// Token: 0x060000D8 RID: 216 RVA: 0x00009778 File Offset: 0x00007978
		public void AddMarkedArea(int offset)
		{
			MarkedSegment markedSegment = this.FindMarkSegment(offset);
			base.Document.UndoStack.Pause();
			string text = base.Document.GetText(markedSegment.StartLine.Offset, markedSegment.EndLine.EndOffset - markedSegment.StartLine.Offset);
			Regex regex = new Regex("^", RegexOptions.IgnoreCase | RegexOptions.Multiline);
			string text2 = regex.Replace(text, "#");
			base.Document.Replace(markedSegment.StartLine.Offset, markedSegment.EndLine.EndOffset - markedSegment.StartLine.Offset, text2);
			base.Document.UndoStack.Resume();
			markedSegment = this.FindMarkSegment(offset);
			this._markedRenderer.AppendMarkedSegment(markedSegment);
			base.TextArea.TextView.InvalidateMeasure();
		}

		// Token: 0x060000D9 RID: 217 RVA: 0x00009846 File Offset: 0x00007A46
		public void ClearMarkArea()
		{
			this._markedRenderer.ClearMarkedSegment();
		}

		// Token: 0x060000DA RID: 218 RVA: 0x00009854 File Offset: 0x00007A54
		public void RemoveMarkedArea(int offset)
		{
			MarkedSegment markedSegment = this.FindMarkSegment(offset);
			base.Document.UndoStack.Pause();
			string text = base.Document.GetText(markedSegment.StartLine.Offset, markedSegment.EndLine.EndOffset - markedSegment.StartLine.Offset);
			Regex regex = new Regex("^#", RegexOptions.IgnoreCase | RegexOptions.Multiline);
			string text2 = regex.Replace(text, "");
			base.Document.Replace(markedSegment.StartLine.Offset, markedSegment.EndLine.EndOffset - markedSegment.StartLine.Offset, text2);
			base.Document.UndoStack.Resume();
			markedSegment = this.FindMarkSegment(offset);
			this._markedRenderer.RemoveMarkedSegment(markedSegment);
			base.TextArea.TextView.InvalidateMeasure();
		}

		// Token: 0x17000017 RID: 23
		// (get) Token: 0x060000DB RID: 219 RVA: 0x00009922 File Offset: 0x00007B22
		// (set) Token: 0x060000DC RID: 220 RVA: 0x00009934 File Offset: 0x00007B34
		public ThemeOptions Theme
		{
			get
			{
				return (ThemeOptions)base.GetValue(CodeTextEditor.ThemeProperty);
			}
			set
			{
				base.SetValue(CodeTextEditor.ThemeProperty, value);
			}
		}

		// Token: 0x060000DD RID: 221 RVA: 0x00009948 File Offset: 0x00007B48
		private static void OnThemeChanged(DependencyObject dp, DependencyPropertyChangedEventArgs e)
		{
			CodeTextEditor codeTextEditor = dp as CodeTextEditor;
			if (codeTextEditor == null)
			{
				return;
			}
			codeTextEditor.LoadSyntaxHighlighting();
		}

		// Token: 0x060000DE RID: 222 RVA: 0x00009968 File Offset: 0x00007B68
		internal void LoadSyntaxHighlighting()
		{
			try
			{
				string text = "pack://application:,,,/CodeEditWindow;component/FGL_Mode_Default.xshd";
				ThemeOptions theme = PreferenceManager.Current.Settings.Theme;
				if (theme == ThemeOptions.Dark)
				{
					text = "pack://application:,,,/CodeEditWindow;component/FGL_Mode_DarkTheme.xshd";
				}
				StreamResourceInfo resourceStream = Application.GetResourceStream(new Uri(text, UriKind.Absolute));
				StreamReader streamReader = new StreamReader(resourceStream.Stream);
				StringReader stringReader = new StringReader(streamReader.ReadToEnd());
				XmlReader xmlReader = XmlReader.Create(stringReader, null);
				base.SyntaxHighlighting = HighlightingLoader.Load(xmlReader, HighlightingManager.Instance);
			}
			catch
			{
				base.SyntaxHighlighting = null;
			}
		}

		// Token: 0x060000DF RID: 223 RVA: 0x000099F4 File Offset: 0x00007BF4
		public new void Dispose()
		{
			base.Dispose();
			if (this.CompletionWindow != null)
			{
				this.CompletionWindow.Close();
			}
			if (this.list != null)
			{
				this.list.CompletionData.Clear();
			}
			foreach (Behavior behavior in Interaction.GetBehaviors(this))
			{
				behavior.Detach();
			}
			EventAggregatorManager.Global.GetEvent<LoadSpecReferFilesEvent>().Unsubscribe(new Action<string>(this.OnBasicDataUpdate));
			EventAggregatorManager.Global.GetEvent<InsertionCodeEvent>().Unsubscribe(new Action<InsertionCodeEventArgs>(this.OnInsertCodeEvent));
			EventAggregatorManager.Global.GetEvent<DiffBlockCopyContentEvent>().Unsubscribe(new Action<DiffCopyInfo>(this.OnDiffBlockCopy));
			EventAggregatorManager.Global.GetEvent<DiffLineCopyContentEvent>().Unsubscribe(new Action<DiffCopyInfo>(this.OnDiffLineCopy));
			EventAggregatorManager.Global.GetEvent<DiffNormalizationSearchEvent>().Unsubscribe(new Action<SearchKeywordEventArgs>(this.NormalizationSearch));
			EventAggregatorManager.Global.GetEvent<RefreshDiffResultEvent>().Unsubscribe(new Action<PackageKey>(this.OnRefreshStardandDiffResult));
			EventAggregatorManager.Global.GetEvent<SaveDiffBaseOnStandardResultEvent>().Unsubscribe(new Action<string>(this.OnSaveDiffStardandResult));
		}

		// Token: 0x060000E0 RID: 224 RVA: 0x00009B34 File Offset: 0x00007D34
		public void AddRenderRegion(int lineNumber, DiffResultSpanStatus type)
		{
			if (this._diffRenderer != null)
			{
				this._diffRenderer.AddRenderRegion(lineNumber, type);
			}
		}

		// Token: 0x060000E1 RID: 225 RVA: 0x00009B4B File Offset: 0x00007D4B
		public void ClearRenderRegion()
		{
			if (this._diffRenderer != null)
			{
				this._diffRenderer.Clear();
			}
		}

		// Token: 0x060000E2 RID: 226 RVA: 0x00009BB8 File Offset: 0x00007DB8
		public void OnDiffBlockCopy(DiffCopyInfo info)
		{
			if (info.ProgramKey.Program != base.ProgramKey.Program)
			{
				return;
			}
			if (info.IsDiffBaseOnStandard)
			{
				return;
			}
			bool flag;
			if (info.FunctionName != "")
			{
				AddPointModel addPointModel = ResourceController.GetInstance().GetProgramInfo(base.ProgramKey).AddPoints.Where<AddPointModel>((AddPointModel a) => a.FunctionNameWithoutParameter.Equals(info.FunctionName) && a.Status == Status.DELETE).ElementAtOrDefault<AddPointModel>(0);
				AddPointModel addPointModel2 = ResourceController.GetInstance().GetProgramInfo(base.ProgramKey).AddPoints.Where<AddPointModel>((AddPointModel a) => a.FunctionNameWithoutParameter.Equals(info.FunctionName) && a.Status != Status.DELETE).ElementAtOrDefault<AddPointModel>(0);
				flag = (addPointModel != null && addPointModel2 != null) || (addPointModel == null && addPointModel2 != null);
			}
			else
			{
				flag = true;
			}
			SegmentObject segmentObject = base.Document.SectionProvider.Find(base.Mode, info.endLine);
			int endLine = info.endLine;
			while (segmentObject == null)
			{
				segmentObject = base.Document.SectionProvider.Find(base.Mode, endLine--);
			}
			if (segmentObject.Parent != null && !segmentObject.IsEditable && segmentObject.Length == 1)
			{
				int num = info.endLine - endLine;
				info.endLine = endLine;
				string[] array = info.SelectContent.Split(new char[] { '\r' });
				string text = string.Empty;
				for (int i = 0; i < array.Count<string>() - num; i++)
				{
					text += array[i];
				}
				info.SelectContent = text;
			}
			bool flag2 = true;
			for (int j = info.startLine; j <= info.endLine; j++)
			{
				SegmentObject segmentObject2 = base.Document.SectionProvider.Find(base.Mode, j);
				if (!segmentObject2.IsEditable)
				{
					MessageBox.Show(Application.Current.FindResource("Message_DiffCopyUnEditable") as string);
					flag2 = false;
					break;
				}
			}
			if (flag && flag2)
			{
				base.TextArea.Document.UndoStack.StartUndoGroup();
				int offset = base.Document.GetLineByNumber(info.startLine).Offset;
				int endOffset = base.Document.GetLineByNumber(info.endLine).EndOffset;
				base.TextArea.Selection = Selection.Create(base.TextArea, offset, endOffset);
				base.TextArea.Selection.ReplaceSelectionWithText(info.SelectContent);
				base.TextArea.Document.UndoStack.EndUndoGroup();
			}
		}

		// Token: 0x060000E3 RID: 227 RVA: 0x00009F14 File Offset: 0x00008114
		public void OnDiffLineCopy(DiffCopyInfo info)
		{
			if (info.ProgramKey.Program != base.ProgramKey.Program)
			{
				return;
			}
			if (info.IsDiffBaseOnStandard)
			{
				return;
			}
			bool flag;
			if (info.FunctionName != "")
			{
				AddPointModel addPointModel = ResourceController.GetInstance().GetProgramInfo(base.ProgramKey).AddPoints.Where<AddPointModel>((AddPointModel a) => a.FunctionNameWithoutParameter.Equals(info.FunctionName) && a.Status == Status.DELETE).ElementAtOrDefault<AddPointModel>(0);
				AddPointModel addPointModel2 = ResourceController.GetInstance().GetProgramInfo(base.ProgramKey).AddPoints.Where<AddPointModel>((AddPointModel a) => a.FunctionNameWithoutParameter.Equals(info.FunctionName) && a.Status != Status.DELETE).ElementAtOrDefault<AddPointModel>(0);
				flag = (addPointModel != null && addPointModel2 != null) || (addPointModel == null && addPointModel2 != null);
			}
			else
			{
				flag = true;
			}
			bool flag2 = true;
			for (int i = info.startLine; i <= info.endLine; i++)
			{
				SegmentObject segmentObject = base.Document.SectionProvider.Find(base.Mode, i);
				if (!segmentObject.IsEditable)
				{
					MessageBox.Show(Application.Current.FindResource("Message_DiffCopyUnEditable") as string);
					flag2 = false;
					break;
				}
			}
			if (flag && flag2)
			{
				base.TextArea.Document.UndoStack.StartUndoGroup();
				int offset = base.Document.GetLineByNumber(info.startLine).Offset;
				int endOffset = base.Document.GetLineByNumber(info.endLine).EndOffset;
				base.TextArea.Selection = Selection.Create(base.TextArea, offset, endOffset);
				base.TextArea.Selection.ReplaceSelectionWithText(info.SelectContent);
				base.TextArea.Document.UndoStack.EndUndoGroup();
			}
		}

		// Token: 0x060000E4 RID: 228 RVA: 0x0000A170 File Offset: 0x00008370
		public void OnDiffFunctionSync(FunctionSyncInfo info)
		{
			XElement xelement = ResourceController.GetInstance().GetProgramInfo(base.ProgramKey).FindDiffSourceByName("function." + info.FunctionName);
			if (xelement == null)
			{
				MessageBox.Show(Application.Current.FindResource("Message_TapSrcIsNull") as string);
				return;
			}
			AddPointModel addPointModel = ResourceController.GetInstance().GetProgramInfo(base.ProgramKey).AddPoints.Where<AddPointModel>((AddPointModel a) => a.FunctionNameWithoutParameter.Equals(info.FunctionName) && a.Status == Status.DELETE).ElementAtOrDefault<AddPointModel>(0);
			AddPointModel addPointModel2 = ResourceController.GetInstance().GetProgramInfo(base.ProgramKey).AddPoints.Where<AddPointModel>((AddPointModel a) => a.FunctionNameWithoutParameter.Equals(info.FunctionName) && a.Status != Status.DELETE).ElementAtOrDefault<AddPointModel>(0);
			if ((addPointModel != null && addPointModel2 != null) || (addPointModel == null && addPointModel2 != null))
			{
				MessageBox.Show(Application.Current.FindResource("Message_FunctionIsExistCanNotSync") as string);
				return;
			}
			CreateFunctionContentArgs createFunctionContentArgs = new CreateFunctionContentArgs();
			createFunctionContentArgs.Content = xelement.Value;
			createFunctionContentArgs.ProgramKey = base.ProgramKey;
			DiffTreeSortInfo diffTreeSortInfo = new DiffTreeSortInfo();
			diffTreeSortInfo.ProgramKey = base.ProgramKey;
			diffTreeSortInfo.NodeName = info.FunctionName;
			diffTreeSortInfo.SortIndex = Convert.ToInt32(xelement.Attribute("order").Value);
			EventController.GetInstance().GetEvent<CreateFunctionEvent>().Publish(createFunctionContentArgs);
			EventAggregatorManager.Global.GetEvent<TreeItemSort>().Publish(diffTreeSortInfo);
		}

		// Token: 0x060000E5 RID: 229 RVA: 0x0000A2E0 File Offset: 0x000084E0
		public void NormalizationSearch(SearchKeywordEventArgs e)
		{
			if (e.Type != DiffType.Normal)
			{
				return;
			}
			if (base.SearchProgramKey != e.ProgramKey)
			{
				base.SearchProgramKey = e.ProgramKey;
			}
			base.SearchKeyword(e, DiffType.Normal);
		}

		// Token: 0x0400003E RID: 62
		private bool _isContentLoaded;

		// Token: 0x0400003F RID: 63
		private bool _isDiffTextViewerModify;

		// Token: 0x04000040 RID: 64
		private TextMarkedRenderer _markedRenderer;

		// Token: 0x04000041 RID: 65
		private DiffRenderer _diffRenderer;

		// Token: 0x04000042 RID: 66
		private GeneroCodeCompletionBinding _codeCompletion;

		// Token: 0x04000043 RID: 67
		private bool _isTyped;

		// Token: 0x04000044 RID: 68
		private Selection _lastCheckSelection;

		// Token: 0x04000045 RID: 69
		private bool _lastCheckEditable;

		// Token: 0x04000046 RID: 70
		private int _lastCaretOffset = -1;

		// Token: 0x04000047 RID: 71
		private bool _lastCheckMarkHard;

		// Token: 0x04000048 RID: 72
		private static RegexOptions regexOptions = RegexOptions.IgnoreCase | RegexOptions.Multiline;

		// Token: 0x04000049 RID: 73
		private static Regex scopeRegex = new Regex("(?<type>^public|^private)", RegexOptions.IgnoreCase | RegexOptions.Multiline);

		// Token: 0x0400004A RID: 74
		private static Regex funcReg = new Regex("^\\s*(function)+\\s+(?<name>\\w+)*(?<parameter>\\([^\\)]*\\))(\\r{0,1})(\\n{0,1})", CodeTextEditor.regexOptions);

		// Token: 0x0400004B RID: 75
		private static Regex endFuncReg = new Regex("(\\r{0,1})(\\n{0,1})^(end function)\\s*$", CodeTextEditor.regexOptions);

		// Token: 0x0400004C RID: 76
		private static Regex dialogReg = new Regex("^\\s*(dialog)+\\s+(?<name>\\w+)*(?<parameter>\\([^\\)]*\\))(\\r{0,1})(\\n{0,1})", CodeTextEditor.regexOptions);

		// Token: 0x0400004D RID: 77
		private static Regex enddialogReg = new Regex("(\\r{0,1})(\\n{0,1})^(end dialog)\\s*$", CodeTextEditor.regexOptions);

		// Token: 0x0400004E RID: 78
		private static Regex reportReg = new Regex("^\\s*(report)+\\s+(?<name>\\w+)*(?<parameter>\\([^\\)]*\\))(\\r{0,1})(\\n{0,1})", CodeTextEditor.regexOptions);

		// Token: 0x0400004F RID: 79
		private static Regex endreportReg = new Regex("(\\r{0,1})(\\n{0,1})^(end report)\\s*$", CodeTextEditor.regexOptions);

		// Token: 0x04000050 RID: 80
		private DesignerCompletionWindow _completionWindow;

		// Token: 0x04000051 RID: 81
		private DesignerInsightWindow _insightWindow;

		// Token: 0x04000052 RID: 82
		private CompletionList list = new CompletionList();

		// Token: 0x04000053 RID: 83
		private ContextMenuCollection _contextMenuCollection;

		// Token: 0x04000054 RID: 84
		public static readonly DependencyProperty ThemeProperty = DependencyProperty.Register("Theme", typeof(ThemeOptions), typeof(CodeTextEditor), new FrameworkPropertyMetadata(ThemeOptions.Default, new PropertyChangedCallback(CodeTextEditor.OnThemeChanged)));

		// Token: 0x02000016 RID: 22
		private class DiffContentStructure
		{
			// Token: 0x17000018 RID: 24
			// (get) Token: 0x060000EA RID: 234 RVA: 0x0000A3E9 File Offset: 0x000085E9
			// (set) Token: 0x060000EB RID: 235 RVA: 0x0000A3F1 File Offset: 0x000085F1
			public string SourceContent { get; set; }

			// Token: 0x17000019 RID: 25
			// (get) Token: 0x060000EC RID: 236 RVA: 0x0000A3FA File Offset: 0x000085FA
			// (set) Token: 0x060000ED RID: 237 RVA: 0x0000A402 File Offset: 0x00008602
			public string TargetContent { get; set; }

			// Token: 0x060000EE RID: 238 RVA: 0x0000A40B File Offset: 0x0000860B
			public DiffContentStructure(string source, string target)
			{
				this.SourceContent = source;
				this.TargetContent = target;
			}
		}

		// Token: 0x02000017 RID: 23
		private class DiffBaseOnStandardStructure
		{
			// Token: 0x1700001A RID: 26
			// (get) Token: 0x060000EF RID: 239 RVA: 0x0000A421 File Offset: 0x00008621
			// (set) Token: 0x060000F0 RID: 240 RVA: 0x0000A429 File Offset: 0x00008629
			public string SourceContent { get; set; }

			// Token: 0x1700001B RID: 27
			// (get) Token: 0x060000F1 RID: 241 RVA: 0x0000A432 File Offset: 0x00008632
			// (set) Token: 0x060000F2 RID: 242 RVA: 0x0000A43A File Offset: 0x0000863A
			public string TargetContent { get; set; }

			// Token: 0x1700001C RID: 28
			// (get) Token: 0x060000F3 RID: 243 RVA: 0x0000A443 File Offset: 0x00008643
			// (set) Token: 0x060000F4 RID: 244 RVA: 0x0000A44B File Offset: 0x0000864B
			public AddPointModel APT { get; set; }

			// Token: 0x060000F5 RID: 245 RVA: 0x0000A454 File Offset: 0x00008654
			public DiffBaseOnStandardStructure(string source, string target, AddPointModel model)
			{
				this.SourceContent = source;
				this.TargetContent = target;
				this.APT = model;
			}
		}
	}
}
