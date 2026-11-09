using System;
using System.Collections.Generic;
using System.Collections.ObjectModel;
using System.Linq;
using System.Text.RegularExpressions;
using System.Windows;
using System.Windows.Controls;
using System.Windows.Controls.Primitives;
using System.Windows.Data;
using System.Windows.Input;
using System.Windows.Interactivity;
using System.Windows.Shapes;
using ICSharpCode.AvalonEdit;
using ICSharpCode.AvalonEdit.Controls;
using ICSharpCode.AvalonEdit.Document;
using ICSharpCode.AvalonEdit.Editing;
using SpecDesigner.CodeEditWindow.Helper;
using SpecDesigner.Controls.Controls;
using SpecDesigner.Controls.Controls.BrushEditor;
using SpecDesigner.Infrastructure;
using SpecDesigner.Infrastructure.Event;
using SpecDesigner.Infrastructure.Model;
using SpecDesignerCommon;
using SpecDesignerCommon.Events;
using SpecDesignerPreference;

namespace SpecDesigner.CodeEditWindow.View
{
	// Token: 0x02000006 RID: 6
	public abstract class BaseTextEditor : TextEditor, IDisposable
	{
		// Token: 0x17000004 RID: 4
		// (get) Token: 0x06000012 RID: 18 RVA: 0x00002442 File Offset: 0x00000642
		// (set) Token: 0x06000013 RID: 19 RVA: 0x0000244A File Offset: 0x0000064A
		public PackageKey ProgramKey { get; protected set; }

		// Token: 0x17000005 RID: 5
		// (get) Token: 0x06000014 RID: 20 RVA: 0x00002453 File Offset: 0x00000653
		// (set) Token: 0x06000015 RID: 21 RVA: 0x0000245B File Offset: 0x0000065B
		public PackageKey SearchProgramKey { get; protected set; }

		// Token: 0x06000016 RID: 22 RVA: 0x00002464 File Offset: 0x00000664
		public BaseTextEditor()
		{
			this._searchRenderer = new SearchResultBackgroundRenderer();
			base.TextArea.TextView.BackgroundRenderers.Add(this._searchRenderer);
			base.Document.TextChanged += this.Document_TextChanged;
			ObservableCollection<UIElement> leftMargins = base.TextArea.LeftMargins;
			leftMargins.Add(new EditableAreaMargin
			{
				Margin = new Thickness(2.0, 0.0, 0.0, 0.0)
			});
			Line line = (Line)DottedLineMargin.Create();
			Binding binding = new Binding("LineNumbersForeground")
			{
				Source = this
			};
			line.SetBinding(Shape.StrokeProperty, binding);
			leftMargins.Add(line);
			Binding binding2 = new Binding("Foreground")
			{
				Source = PreferenceManager.Current.Settings,
				UpdateSourceTrigger = UpdateSourceTrigger.PropertyChanged,
				Mode = BindingMode.TwoWay,
				Converter = new BrushConverter()
			};
			base.TextArea.TextView.SetBinding(Control.ForegroundProperty, binding2);
			Binding binding3 = new Binding("FontSize")
			{
				Source = PreferenceManager.Current.Settings,
				UpdateSourceTrigger = UpdateSourceTrigger.PropertyChanged,
				Mode = BindingMode.TwoWay
			};
			base.SetBinding(Control.FontSizeProperty, binding3);
			Binding binding4 = new Binding("FontFamily")
			{
				Source = PreferenceManager.Current.Settings,
				UpdateSourceTrigger = UpdateSourceTrigger.PropertyChanged,
				Mode = BindingMode.TwoWay
			};
			base.SetBinding(Control.FontFamilyProperty, binding4);
			Binding binding5 = new Binding("Background")
			{
				Source = PreferenceManager.Current.Settings,
				UpdateSourceTrigger = UpdateSourceTrigger.PropertyChanged,
				Mode = BindingMode.TwoWay,
				Converter = new BrushConverter()
			};
			base.SetBinding(Control.BackgroundProperty, binding5);
			base.HorizontalScrollBarVisibility = (PreferenceManager.Current.Settings.WordWrap ? ScrollBarVisibility.Disabled : ScrollBarVisibility.Auto);
			base.ShowLineNumbers = true;
			base.Options.ConvertTabsToSpaces = true;
			if (PreferenceManager.Current.Settings.ShowHint)
			{
				ToolTipService.SetToolTip(this, new ToolTip
				{
					PlacementTarget = this,
					Placement = PlacementMode.Left
				});
				ToolTipService.SetInitialShowDelay(this, 10);
				Interaction.GetBehaviors(this).Add(new ShowHintBehavior());
			}
			EventAggregatorManager.Global.GetEvent<SearchKeywordEvent>().Subscribe(new Action<SearchKeywordEventArgs>(this.SearchKeyword));
			EventAggregatorManager.Global.GetEvent<SearchResultInfoSelectedEvent>().Subscribe(new Action<SearchResultInfo>(this.SearchResultInfoSelected));
			EventAggregatorManager.Global.GetEvent<SearchFromCaret>().Subscribe(new Action<SearchKeywordEventArgs>(this.SearchResultFormCaret));
			EventAggregatorManager.Global.GetEvent<ReplaceKeywordEvent>().Subscribe(new Action<SearchKeywordEventArgs>(this.SearchKeyword));
			EventAggregatorManager.Global.GetEvent<ReplaceResultEvent>().Subscribe(new Action<ReplaceResultInfo>(this.ReplaceResultInfoSelected));
			EventAggregatorManager.Global.GetEvent<ReplaceEvent>().Subscribe(new Action<ReplaceResultInfo>(this.ReplaceOne));
			EventAggregatorManager.Global.GetEvent<ReplaceAllEvent>().Subscribe(new Action<ReplaceResultInfo>(this.ReplaceAll));
			EventAggregatorManager.Global.GetEvent<ReplaceAllFromCaret>().Subscribe(new Action<ReplaceResultInfo>(this.ReplaceAllFromCaret));
			base.CommandBindings.Add(new CommandBinding(CodeMenuCommands.FindNextCommand, new ExecutedRoutedEventHandler(this.ExecutedFindNext)));
			base.CommandBindings.Add(new CommandBinding(CodeMenuCommands.FindPreviousCommand, new ExecutedRoutedEventHandler(this.ExecutedFindPrevious)));
		}

		// Token: 0x06000017 RID: 23 RVA: 0x000027F8 File Offset: 0x000009F8
		private void ExecutedFindNext(object sender, ExecutedRoutedEventArgs e)
		{
			e.Handled = true;
			SearchKeywordEventArgs e2 = new SearchKeywordEventArgs(this.ProgramKey, this.lastSearchKey, this.lastIsMatchCase)
			{
				IsRegExMode = this.lastIsRegExMode,
				IsSearchFromCaret = true,
				Direction = SearchDirection.Next,
				IsLoop = true
			};
			this.SearchKeyword(e2, false);
		}

		// Token: 0x06000018 RID: 24 RVA: 0x00002850 File Offset: 0x00000A50
		private void ExecutedFindPrevious(object sender, ExecutedRoutedEventArgs e)
		{
			e.Handled = true;
			SearchKeywordEventArgs e2 = new SearchKeywordEventArgs(this.ProgramKey, this.lastSearchKey, this.lastIsMatchCase)
			{
				IsRegExMode = this.lastIsRegExMode,
				IsSearchFromCaret = true,
				Direction = SearchDirection.Previous,
				IsLoop = true
			};
			this.SearchKeyword(e2, false);
		}

		// Token: 0x06000019 RID: 25 RVA: 0x000028A7 File Offset: 0x00000AA7
		public override void OnApplyTemplate()
		{
			base.OnApplyTemplate();
			this._scrollViewer = base.Template.FindName("PART_ScrollViewer", this) as ScrollViewerEnhanced;
		}

		// Token: 0x0600001A RID: 26 RVA: 0x000028CB File Offset: 0x00000ACB
		private void Document_TextChanged(object sender, EventArgs e)
		{
			this.ClearSearchResult();
		}

		// Token: 0x0600001B RID: 27 RVA: 0x000028D4 File Offset: 0x00000AD4
		public void OnNextSearch()
		{
			string text = this.lastSearchKey;
			int caretOffset = base.CaretOffset;
			StringComparison stringComparison = (this.lastIsMatchCase ? StringComparison.CurrentCulture : StringComparison.InvariantCultureIgnoreCase);
			int num = base.Text.IndexOf(text, caretOffset, stringComparison);
			if (num < 0)
			{
				return;
			}
			base.Select(num, text.Length);
		}

		// Token: 0x0600001C RID: 28 RVA: 0x00002920 File Offset: 0x00000B20
		protected void SearchKeyword(SearchKeywordEventArgs e, DiffType type)
		{
			this.type = type;
			this.SearchKeyword(e, true);
			NormalizationSearchResult currentSearchResult = ResourceController.GetInstance().GetProgramInfo(this.SearchProgramKey).CurrentSearchResult;
			List<NormalizationSearchResult> results = ResourceController.GetInstance().GetProgramInfo(this.SearchProgramKey).Results;
			if (currentSearchResult == null && results.Count<NormalizationSearchResult>() > 0)
			{
				ResourceController.GetInstance().GetProgramInfo(this.SearchProgramKey).CurrentSearchResult = ResourceController.GetInstance().GetProgramInfo(this.SearchProgramKey).Results[0];
			}
		}

		// Token: 0x0600001D RID: 29 RVA: 0x000029A4 File Offset: 0x00000BA4
		protected void SearchKeyword(SearchKeywordEventArgs e)
		{
			this.SearchKeyword(e, true);
		}

		// Token: 0x0600001E RID: 30 RVA: 0x000029B0 File Offset: 0x00000BB0
		private void SearchKeyword(SearchKeywordEventArgs e, bool isPublishEvent)
		{
			if (e.ProgramKey != null && (e.ProgramKey.Program != this.ProgramKey.Program || e.ProgramKey.PackType != this.ProgramKey.PackType))
			{
				return;
			}
			string text = string.Empty;
			int num = (e.IsSearchFromCaret ? ((base.SelectionLength > 0) ? base.SelectionStart : base.CaretOffset) : 0);
			if (!e.IsSearchFromCaret)
			{
				text = base.Text;
			}
			else if (e.SelectedOnly)
			{
				text = base.TextArea.Selection.GetText();
			}
			else
			{
				switch (e.Direction)
				{
				case SearchDirection.Forward:
					text = base.Document.GetText(num, base.Text.Length - num);
					break;
				case SearchDirection.Backward:
					num = 0;
					text = base.Document.GetText(0, num);
					break;
				case SearchDirection.Next:
				{
					num = ((base.SelectionLength > 0) ? (base.SelectionStart + base.SelectionLength) : base.CaretOffset);
					int num2;
					if (num <= 1)
					{
						num2 = num;
					}
					else
					{
						num = (num2 = num) - 1;
					}
					num = num2;
					num = ((base.Document.TextLength - 1 == num && e.IsLoop) ? 0 : num);
					text = base.Document.GetText(num, (base.Text.Length - num - 1 > 0) ? (base.Text.Length - num - 1) : (base.Text.Length - num));
					break;
				}
				case SearchDirection.Previous:
					num = ((num == 0 && e.IsLoop) ? base.Document.TextLength : num);
					text = base.Document.GetText(0, (num - 1 > 0) ? (num - 1) : 0);
					num = 0;
					break;
				}
			}
			this.lastSearchKey = e.Keyword;
			this.lastIsMatchCase = e.IsMatchCase;
			this.lastIsRegExMode = e.IsRegExMode;
			this.ClearSearchResult();
			if (e.IsRegExMode)
			{
				RegexOptions regexOptions = RegexOptions.Multiline | RegexOptions.IgnorePatternWhitespace;
				if (!e.IsMatchCase)
				{
					regexOptions |= RegexOptions.IgnoreCase;
				}
				try
				{
					Regex regex = new Regex(e.Keyword, regexOptions);
					MatchCollection matchCollection = regex.Matches(text);
					if (matchCollection.Count == 0 && e.IsLoop)
					{
						base.CaretOffset = ((e.Direction == SearchDirection.Forward || e.Direction == SearchDirection.Next) ? 0 : base.Document.Text.Length);
						base.TextArea.ClearSelection();
						e.IsLoop = false;
						this.SearchKeyword(e, isPublishEvent);
						return;
					}
					int num3 = ((e.Direction == SearchDirection.Forward || e.Direction == SearchDirection.Next) ? 0 : (matchCollection.Count - 1));
					while (num3 < matchCollection.Count && num3 >= 0)
					{
						Match match = matchCollection[num3];
						int num4 = match.Index + num;
						DocumentLine lineByOffset = base.Document.GetLineByOffset(num4);
						SearchResultInfo searchResultInfo = new SearchResultInfo();
						searchResultInfo.Mode = SearchMode.Regex;
						searchResultInfo.Memo = lineByOffset.LineNumber;
						searchResultInfo.Keyword = match.Value;
						searchResultInfo.Key = num4.ToString();
						searchResultInfo.ProgramKey = this.ProgramKey;
						searchResultInfo.SourceType = this.ProgramKey.PackType;
						if (base.Document.SectionProvider.CanInsert(num4))
						{
							goto IL_0335;
						}
						if (!e.EditableAreaOnly)
						{
							searchResultInfo.IsEditable = false;
							goto IL_0335;
						}
						IL_04B7:
						num3 = ((e.Direction == SearchDirection.Forward || e.Direction == SearchDirection.Next) ? (num3 + 1) : (num3 - 1));
						continue;
						IL_0335:
						if (isPublishEvent)
						{
							searchResultInfo.Match = base.Document.GetText(new TextSegment
							{
								StartOffset = lineByOffset.Offset,
								EndOffset = lineByOffset.EndOffset
							});
							if (!SettingManager.Get().GetTzpManger(this.SearchProgramKey).IsMajorAbnormal)
							{
								EventAggregatorManager.Global.GetEvent<SearchKeywordResultEvent>().Publish(searchResultInfo);
							}
						}
						else if (this.replaceList != null)
						{
							searchResultInfo.Memo = match.Index;
							this.replaceList.Add(searchResultInfo);
						}
						if (e.IsSearchFromCaret && e.ReplaceAs == null && (((e.Direction == SearchDirection.Previous || e.Direction == SearchDirection.Backward) && num3 == matchCollection.Count - 1) || ((e.Direction == SearchDirection.Next || e.Direction == SearchDirection.Previous) && num3 == 0)))
						{
							base.Select(num4, match.Value.Length);
							break;
						}
						SearchResult searchResult = new SearchResult
						{
							StartOffset = num4,
							Length = match.Value.Length
						};
						this.AddSearchResult(searchResult);
						if (SettingManager.Get().GetTzpManger(this.SearchProgramKey).IsDiff && this.ProgramKey.Memo == MemoType.CodeDiff)
						{
							NormalizationSearchResult normalizationSearchResult = new NormalizationSearchResult
							{
								StartOffset = num4,
								Length = match.Value.Length,
								LineNumber = lineByOffset.LineNumber
							};
							ResourceController.GetInstance().GetProgramInfo(this.SearchProgramKey).AddNormalizationSearchResult(normalizationSearchResult);
							goto IL_04B7;
						}
						goto IL_04B7;
					}
					return;
				}
				catch (Exception)
				{
					DesignerMessageBox.Show(Application.Current.FindResource("Message_RegFormatError") as string);
					return;
				}
			}
			StringComparison stringComparison = (e.IsMatchCase ? StringComparison.CurrentCulture : StringComparison.InvariantCultureIgnoreCase);
			int i = 0;
			while (i < text.Length)
			{
				int num5 = ((e.Direction == SearchDirection.Forward || e.Direction == SearchDirection.Next) ? text.IndexOf(e.Keyword, i, stringComparison) : text.LastIndexOf(e.Keyword, stringComparison));
				if (e.SelectedOnly && base.SelectionLength > 0 && num5 > base.SelectionStart + base.SelectionLength)
				{
					return;
				}
				if (num5 < 0)
				{
					if (e.IsLoop)
					{
						base.CaretOffset = 0;
						base.TextArea.ClearSelection();
						e.IsLoop = false;
						this.SearchKeyword(e, isPublishEvent);
						return;
					}
					return;
				}
				else
				{
					int num6 = num5 + num;
					DocumentLine lineByOffset2 = base.Document.GetLineByOffset(num6);
					SearchResultInfo searchResultInfo2 = new SearchResultInfo();
					if (!base.Document.SectionProvider.CanInsert(num6))
					{
						if (e.EditableAreaOnly)
						{
							i = num5 + 1;
							continue;
						}
						searchResultInfo2.IsEditable = false;
					}
					searchResultInfo2.Memo = lineByOffset2.LineNumber;
					searchResultInfo2.Keyword = e.Keyword;
					searchResultInfo2.Key = num6.ToString();
					searchResultInfo2.ProgramKey = this.ProgramKey;
					searchResultInfo2.SourceType = this.ProgramKey.PackType;
					if (isPublishEvent)
					{
						searchResultInfo2.Match = base.Document.GetText(new TextSegment
						{
							StartOffset = lineByOffset2.Offset,
							EndOffset = lineByOffset2.EndOffset
						});
						EventAggregatorManager.Global.GetEvent<SearchKeywordResultEvent>().Publish(searchResultInfo2);
					}
					else if (this.replaceList != null)
					{
						searchResultInfo2.Memo = num6;
						this.replaceList.Add(searchResultInfo2);
					}
					if (e.IsSearchFromCaret && e.ReplaceAs == null)
					{
						base.Select(num6, e.Keyword.Length);
						return;
					}
					SearchResult searchResult2 = new SearchResult
					{
						StartOffset = num6,
						Length = e.Keyword.Length
					};
					this.AddSearchResult(searchResult2);
					i = num5 + 1;
				}
			}
		}

		// Token: 0x0600001F RID: 31 RVA: 0x000030EC File Offset: 0x000012EC
		protected void SearchResultFormCaret(SearchKeywordEventArgs args)
		{
			if (args.ProgramKey != null && this.ProgramKey != args.ProgramKey)
			{
				return;
			}
			this.SearchKeyword(args, args.IsLoop);
		}

		// Token: 0x06000020 RID: 32 RVA: 0x00003120 File Offset: 0x00001320
		protected void SearchResultInfoSelected(SearchResultInfo info)
		{
			if (info.ProgramKey.Equals(this.ProgramKey))
			{
				int num;
				if (!int.TryParse(info.Key, out num))
				{
					TreeItem treeItem = (from t in ResourceController.GetInstance().GetProgramInfo(this.ProgramKey).Find(info.Key)
						select (t)).ElementAtOrDefault<TreeItem>(0);
					if (treeItem != null)
					{
						treeItem.PublishSelectedEvent();
					}
					return;
				}
				DocumentLine lineByOffset = base.Document.GetLineByOffset(num);
				SearchResult searchResult = new SearchResult
				{
					StartOffset = num
				};
				searchResult.Length = info.Keyword.Length;
				this.SelectSearchResult(searchResult);
				base.ScrollTo(lineByOffset.LineNumber);
				base.CaretOffset = num;
			}
		}

		// Token: 0x06000021 RID: 33 RVA: 0x0000321C File Offset: 0x0000141C
		protected void ReplaceResultInfoSelected(ReplaceResultInfo info)
		{
			if (this.ProgramKey != info.SearchInformation.ProgramKey && info.SearchInformation.ProgramKey != null)
			{
				return;
			}
			if (info.SearchInformation.SourceType == this.ProgramKey.PackType && info.SearchInformation.ProgramKey == this.ProgramKey)
			{
				int num;
				if (!int.TryParse(info.SearchInformation.Key, out num))
				{
					TreeItem treeItem = (from t in ResourceController.GetInstance().GetProgramInfo(this.ProgramKey).Find(info.SearchInformation.Key)
						select (t)).ElementAtOrDefault<TreeItem>(0);
					if (treeItem != null)
					{
						treeItem.PublishSelectedEvent();
					}
					return;
				}
				this.ClearSearchResult();
				DocumentLine lineByOffset = base.Document.GetLineByOffset(num);
				SegmentObject segment = base.Document.SectionProvider.Find(base.Mode, lineByOffset.LineNumber);
				if (segment.Parent != null && segment is EditObject && !segment.IsEditable)
				{
					AddPointModel addPointModel = ResourceController.GetInstance().GetProgramInfo(this.ProgramKey).AddPoints.Where<AddPointModel>((AddPointModel ap) => ap.ID == segment.ID && (ap.Status & Status.DELETE) == Status.NULL).ElementAtOrDefault<AddPointModel>(0);
					ModifyInfomation modifyInfomation = new ModifyInfomation();
					modifyInfomation.ModifyType = ModifyInfomation.ModifyTypeEnum.Name;
					modifyInfomation.ProgramKey = this.ProgramKey;
					modifyInfomation.Source = addPointModel.Name;
					modifyInfomation.Modified = addPointModel.FunctionName.Replace(info.SearchInformation.Keyword, info.ReplaceAs);
					ResourceController.GetInstance().GetProgramInfo(this.ProgramKey).Modify(modifyInfomation);
					EventController.GetInstance().GetEvent<ModifyFunctionEvent>().Publish(modifyInfomation);
				}
				else
				{
					base.Document.Replace(num, info.SearchInformation.Keyword.Length, info.ReplaceAs);
					base.ScrollTo(lineByOffset.LineNumber);
					base.CaretOffset = num;
				}
				EventAggregatorManager.Get(this.ProgramKey).GetEvent<CodeChangedEvent>().Publish(this.ProgramKey);
			}
		}

		// Token: 0x06000022 RID: 34 RVA: 0x00003468 File Offset: 0x00001668
		protected void ReplaceOne(ReplaceResultInfo info)
		{
			PackageKey packageKey = Application.Current.MainWindow.Tag as PackageKey;
			if (packageKey != null && packageKey == this.ProgramKey)
			{
				if (this.replaceList == null)
				{
					this.replaceList = new List<SearchResultInfo>();
				}
				if (this.replaceList != null && this.replaceList.Count > 1 && base.SelectionLength > 0 && base.TextArea.Selection.GetText() == info.Keyword && this.replaceList[0].Memo == base.SelectionStart)
				{
					base.Document.UndoStack.StartUndoGroup();
					SearchResultInfo searchResultInfo = this.replaceList[0];
					this.ReplaceResultInfoSelected(new ReplaceResultInfo(searchResultInfo)
					{
						ReplaceAs = info.ReplaceAs
					});
					base.Document.UndoStack.EndUndoGroup();
					this.replaceList.RemoveAt(0);
				}
				else
				{
					this.replaceList.Clear();
					this.SearchKeyword(new SearchKeywordEventArgs(this.ProgramKey, info.Keyword, info.IsMatchCase, true, info.SelectedOnly)
					{
						IsSearchFromCaret = (base.TextArea.Selection.Length > 0),
						ReplaceAs = ((info.ReplaceAs.Length > 0) ? info.ReplaceAs : string.Empty)
					}, false);
					if (this.replaceList.Count > 0)
					{
						base.Document.UndoStack.StartUndoGroup();
						SearchResultInfo searchResultInfo2 = this.replaceList[0];
						this.ReplaceResultInfoSelected(new ReplaceResultInfo(searchResultInfo2)
						{
							ReplaceAs = info.ReplaceAs
						});
						base.Document.UndoStack.EndUndoGroup();
						this.replaceList.RemoveAt(0);
					}
				}
				this.AdjustReplaceListOffset(info.ReplaceAs.Length, info.Keyword.Length);
				if (this.replaceList.Count > 0)
				{
					int memo = this.replaceList[0].Memo;
					base.Select(memo, info.Keyword.Length);
				}
			}
		}

		// Token: 0x06000023 RID: 35 RVA: 0x00003698 File Offset: 0x00001898
		private void AdjustReplaceListOffset(int replaceAsLength, int keywordLength)
		{
			if (this.replaceList == null)
			{
				return;
			}
			foreach (SearchResultInfo searchResultInfo in this.replaceList)
			{
				searchResultInfo.Memo += replaceAsLength - keywordLength;
				searchResultInfo.Key = searchResultInfo.Memo.ToString();
			}
		}

		// Token: 0x06000024 RID: 36 RVA: 0x00003714 File Offset: 0x00001914
		protected void ReplaceAll(ReplaceResultInfo info)
		{
			PackageKey packageKey = Application.Current.MainWindow.Tag as PackageKey;
			if (packageKey != null && packageKey == this.ProgramKey)
			{
				if (this.replaceList == null)
				{
					this.replaceList = new List<SearchResultInfo>();
				}
				this.replaceList.Clear();
				SearchKeywordEventArgs e = new SearchKeywordEventArgs(this.ProgramKey, info.Keyword, info.IsMatchCase, true, info.SelectedOnly);
				this.SearchKeyword(e, false);
				base.Document.UndoStack.StartUndoGroup();
				int num = info.ReplaceAs.Length - info.Keyword.Length;
				for (int i = 0; i < this.replaceList.Count; i++)
				{
					int num2 = ((i > 0) ? (int.Parse(this.replaceList[i].Key) + num * i) : int.Parse(this.replaceList[i].Key));
					SearchResultInfo searchResultInfo = this.replaceList[i];
					searchResultInfo.Key = num2.ToString();
					this.ReplaceResultInfoSelected(new ReplaceResultInfo(searchResultInfo)
					{
						ReplaceAs = info.ReplaceAs
					});
				}
				base.Document.UndoStack.EndUndoGroup();
			}
		}

		// Token: 0x06000025 RID: 37 RVA: 0x00003868 File Offset: 0x00001A68
		protected void ReplaceAllFromCaret(ReplaceResultInfo Info)
		{
			base.Document.UndoStack.StartUndoGroup();
			for (int i = Info.startLine; i <= Info.endLine; i++)
			{
				SegmentObject segmentObject = base.Document.SectionProvider.Find(base.Mode, i);
				if (segmentObject != null && segmentObject.IsEditable && segmentObject is EditObject)
				{
					DocumentLine documentLine = base.Document.GetLineByNumber(i);
					string text = base.TextArea.Document.GetText(documentLine.Offset, documentLine.Length);
					while (Regex.IsMatch(text, Info.SearchInformation.Keyword.ToString()))
					{
						int num = documentLine.Offset + text.IndexOf(Info.SearchInformation.Keyword);
						int num2 = num + Info.SearchInformation.Keyword.Length;
						base.TextArea.Selection = Selection.Create(base.TextArea, num, num2);
						base.TextArea.Selection.ReplaceSelectionWithText(Info.ReplaceAs);
						documentLine = base.Document.GetLineByNumber(i);
						text = base.TextArea.Document.GetText(num + Info.ReplaceAs.Length, documentLine.Length - (num2 - (Info.SearchInformation.Keyword.Length - Info.ReplaceAs.Length) - documentLine.Offset));
					}
				}
				else if (segmentObject != null)
				{
					i = segmentObject.EndOffset;
				}
			}
			base.Document.UndoStack.EndUndoGroup();
		}

		// Token: 0x06000026 RID: 38 RVA: 0x00003A00 File Offset: 0x00001C00
		public void ClearSearchResult()
		{
			if (this._searchRenderer != null && this._searchRenderer.Results != null)
			{
				this._searchRenderer.Results.Clear();
			}
			base.TextArea.TextView.InvalidateMeasure();
			if (this._scrollViewer != null)
			{
				this._scrollViewer.ClearBookmark(MarkPurpose.Search);
			}
		}

		// Token: 0x06000027 RID: 39 RVA: 0x00003A56 File Offset: 0x00001C56
		public void AddSearchResult(SearchResult result)
		{
			this._searchRenderer.Results.Add(result);
		}

		// Token: 0x06000028 RID: 40 RVA: 0x00003A69 File Offset: 0x00001C69
		public void SelectSearchResult(SearchResult result)
		{
			base.Select(result.StartOffset, result.Length);
		}

		// Token: 0x06000029 RID: 41 RVA: 0x00003A7D File Offset: 0x00001C7D
		public void Dispose()
		{
			base.CommandBindings.Clear();
			base.Document.TextChanged -= this.Document_TextChanged;
			this.UnsubscribeEvent();
		}

		// Token: 0x0600002A RID: 42 RVA: 0x00003AA8 File Offset: 0x00001CA8
		protected void UnsubscribeEvent()
		{
			EventAggregatorManager.Global.GetEvent<SearchKeywordEvent>().Unsubscribe(new Action<SearchKeywordEventArgs>(this.SearchKeyword));
			EventAggregatorManager.Global.GetEvent<SearchResultInfoSelectedEvent>().Unsubscribe(new Action<SearchResultInfo>(this.SearchResultInfoSelected));
			EventAggregatorManager.Global.GetEvent<SearchFromCaret>().Unsubscribe(new Action<SearchKeywordEventArgs>(this.SearchResultFormCaret));
			EventAggregatorManager.Global.GetEvent<ReplaceKeywordEvent>().Unsubscribe(new Action<SearchKeywordEventArgs>(this.SearchKeyword));
			EventAggregatorManager.Global.GetEvent<ReplaceResultEvent>().Unsubscribe(new Action<ReplaceResultInfo>(this.ReplaceResultInfoSelected));
			EventAggregatorManager.Global.GetEvent<ReplaceEvent>().Unsubscribe(new Action<ReplaceResultInfo>(this.ReplaceOne));
			EventAggregatorManager.Global.GetEvent<ReplaceAllEvent>().Unsubscribe(new Action<ReplaceResultInfo>(this.ReplaceAll));
			EventAggregatorManager.Global.GetEvent<ReplaceAllFromCaret>().Unsubscribe(new Action<ReplaceResultInfo>(this.ReplaceAllFromCaret));
		}

		// Token: 0x0400000B RID: 11
		private string lastSearchKey = string.Empty;

		// Token: 0x0400000C RID: 12
		private bool lastIsMatchCase;

		// Token: 0x0400000D RID: 13
		private bool lastIsRegExMode;

		// Token: 0x0400000E RID: 14
		private DiffType type;

		// Token: 0x0400000F RID: 15
		private SearchResultBackgroundRenderer _searchRenderer;

		// Token: 0x04000010 RID: 16
		private ScrollViewerEnhanced _scrollViewer;

		// Token: 0x04000011 RID: 17
		private List<SearchResultInfo> replaceList;
	}
}
