using System;
using System.Collections.Generic;
using System.Linq;
using System.Windows;
using System.Windows.Interactivity;
using System.Windows.Media;
using ICSharpCode.AvalonEdit;
using ICSharpCode.AvalonEdit.Controls;
using SpecDesigner.CodeEditWindow.View;
using SpecDesigner.Controls.Controls;
using SpecDesigner.Infrastructure;
using SpecDesigner.Infrastructure.Event;
using SpecDesigner.Infrastructure.Model;
using SpecDesignerCommon;

namespace SpecDesigner.CodeEditWindow.Helper
{
	// Token: 0x0200000A RID: 10
	public class BookmarkTrackerBehavior : Behavior<CodeTextEditor>
	{
		// Token: 0x06000050 RID: 80 RVA: 0x00004C15 File Offset: 0x00002E15
		public BookmarkTrackerBehavior()
		{
			EventController.GetInstance().GetEvent<ADPStatusUpdateEvent>().Subscribe(new Action<PackageKey>(this.OnADPStatusUpdate));
		}

		// Token: 0x06000051 RID: 81 RVA: 0x00004C3C File Offset: 0x00002E3C
		protected override void OnAttached()
		{
			base.OnAttached();
			this._editor = base.AssociatedObject;
			this._programKey = this._editor.ProgramKey;
			base.AssociatedObject.Loaded += this.AssociatedObject_Loaded;
			this._editor.ModeChanged += this._editor_ModeChanged;
		}

		// Token: 0x06000052 RID: 82 RVA: 0x00004C9A File Offset: 0x00002E9A
		protected override void OnDetaching()
		{
			base.OnDetaching();
			this._editor.ModeChanged -= this._editor_ModeChanged;
			if (this._scrollViewer != null)
			{
				this._scrollViewer.ClearBookmark(MarkPurpose.Modified);
			}
		}

		// Token: 0x06000053 RID: 83 RVA: 0x00004CCD File Offset: 0x00002ECD
		private void _editor_ModeChanged(object sender, EventArgs e)
		{
			this.CreateBookmark();
		}

		// Token: 0x06000054 RID: 84 RVA: 0x00004CD8 File Offset: 0x00002ED8
		private void AssociatedObject_Loaded(object sender, RoutedEventArgs e)
		{
			base.AssociatedObject.Loaded -= this.AssociatedObject_Loaded;
			this._scrollViewer = this._editor.Template.FindName("PART_ScrollViewer", base.AssociatedObject) as ScrollViewerEnhanced;
			this.CreateBookmark();
		}

		// Token: 0x06000055 RID: 85 RVA: 0x00004D28 File Offset: 0x00002F28
		private void OnADPStatusUpdate(PackageKey key)
		{
			if (key != this._programKey)
			{
				return;
			}
			try
			{
				this.CreateBookmark();
			}
			catch
			{
				DesignerMessageBox.Show("Oops! Can't update bookmarks");
			}
		}

		// Token: 0x06000056 RID: 86 RVA: 0x00004D9C File Offset: 0x00002F9C
		private void CreateBookmark()
		{
			if (this._scrollViewer == null || null == this._programKey || ResourceController.GetInstance().GetProgramInfo(this._programKey) == null)
			{
				return;
			}
			this._scrollViewer.ClearBookmark(MarkPurpose.Modified);
			switch (this._editor.Mode)
			{
			case ContentType.ADP:
			{
				IEnumerable<AddPointModel> enumerable = ResourceController.GetInstance().GetProgramInfo(this._programKey).AddPoints.Where<AddPointModel>((AddPointModel adp) => (adp.Status & Status.MODIFY) == Status.MODIFY && (adp.Status & Status.DELETE) != Status.DELETE);
				using (IEnumerator<AddPointModel> enumerator = enumerable.GetEnumerator())
				{
					while (enumerator.MoveNext())
					{
						AddPointModel addPointModel = enumerator.Current;
						BasicShapeScrollBarBookmark basicShapeScrollBarBookmark = null;
						try
						{
							int offset = this._editor.Document.SectionProvider.Find(addPointModel.ID).Offset;
							double visualTopByDocumentLine = this._editor.TextArea.TextView.GetVisualTopByDocumentLine(offset);
							int endOffset = this._editor.Document.SectionProvider.Find(addPointModel.ID).EndOffset;
							int num = (int)((double)(endOffset - offset) * this._editor.TextArea.TextView.DefaultLineHeight);
							basicShapeScrollBarBookmark = new BasicShapeScrollBarBookmark("ADP", visualTopByDocumentLine, ScrollBarBookmarkAlignment.LeftOrTop, num, 3, ScrollbarBookmarkShape.Rectangle, Colors.Red, true, false, null);
							basicShapeScrollBarBookmark.Purpose = MarkPurpose.Modified;
						}
						catch
						{
						}
						finally
						{
							if (basicShapeScrollBarBookmark != null)
							{
								this._scrollViewer.AddBookmark(basicShapeScrollBarBookmark);
							}
						}
					}
					return;
				}
				break;
			}
			case ContentType.SEC:
				break;
			default:
				return;
			}
			IEnumerable<SectionModel> enumerable2 = ResourceController.GetInstance().GetProgramInfo(this._programKey).Sections.Where<SectionModel>((SectionModel sec) => sec.Status == "u");
			foreach (SectionModel sectionModel in enumerable2)
			{
				BasicShapeScrollBarBookmark basicShapeScrollBarBookmark2 = null;
				try
				{
					int offset2 = this._editor.Document.SectionProvider.Find(ContentType.SEC, sectionModel.ID).Offset;
					double visualTopByDocumentLine2 = this._editor.TextArea.TextView.GetVisualTopByDocumentLine(offset2);
					int endOffset2 = this._editor.Document.SectionProvider.Find(ContentType.SEC, sectionModel.ID).EndOffset;
					int num2 = (int)((double)(endOffset2 - offset2) * this._editor.TextArea.TextView.DefaultLineHeight);
					basicShapeScrollBarBookmark2 = new BasicShapeScrollBarBookmark("SEC", visualTopByDocumentLine2, ScrollBarBookmarkAlignment.LeftOrTop, num2, 3, ScrollbarBookmarkShape.Rectangle, Colors.Red, true, false, null);
					basicShapeScrollBarBookmark2.Purpose = MarkPurpose.Modified;
				}
				catch
				{
				}
				finally
				{
					if (basicShapeScrollBarBookmark2 != null)
					{
						this._scrollViewer.AddBookmark(basicShapeScrollBarBookmark2);
					}
				}
			}
		}

		// Token: 0x04000020 RID: 32
		private ScrollViewerEnhanced _scrollViewer;

		// Token: 0x04000021 RID: 33
		private CodeTextEditor _editor;

		// Token: 0x04000022 RID: 34
		private PackageKey _programKey;
	}
}
