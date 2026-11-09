using System;
using System.Collections.Generic;
using System.Collections.ObjectModel;
using System.Linq;
using System.Windows.Media;
using ICSharpCode.AvalonEdit.Rendering;

namespace SpecDesigner.CodeEditWindow.Helper
{
	// Token: 0x02000011 RID: 17
	internal class SearchResultBackgroundRenderer : IBackgroundRenderer
	{
		// Token: 0x17000012 RID: 18
		// (get) Token: 0x0600007F RID: 127 RVA: 0x00005889 File Offset: 0x00003A89
		// (set) Token: 0x06000080 RID: 128 RVA: 0x00005891 File Offset: 0x00003A91
		public List<SearchResult> Results { get; set; }

		// Token: 0x17000013 RID: 19
		// (get) Token: 0x06000081 RID: 129 RVA: 0x0000589A File Offset: 0x00003A9A
		public KnownLayer Layer
		{
			get
			{
				return KnownLayer.Selection;
			}
		}

		// Token: 0x06000082 RID: 130 RVA: 0x0000589D File Offset: 0x00003A9D
		public SearchResultBackgroundRenderer()
		{
			this.Results = new List<SearchResult>();
		}

		// Token: 0x06000083 RID: 131 RVA: 0x000058B0 File Offset: 0x00003AB0
		private static Brush CreateBackgroundBrush()
		{
			return new SolidColorBrush(Colors.Yellow);
		}

		// Token: 0x06000084 RID: 132 RVA: 0x000058CC File Offset: 0x00003ACC
		public void Draw(TextView textView, DrawingContext drawingContext)
		{
			if (textView == null)
			{
				throw new ArgumentNullException("textView");
			}
			if (drawingContext == null)
			{
				throw new ArgumentNullException("drawingContext");
			}
			if (this.Results == null || this.Results.Count == 0 || !textView.VisualLinesValid)
			{
				return;
			}
			ReadOnlyCollection<VisualLine> visualLines = textView.VisualLines;
			if (visualLines.Count == 0)
			{
				return;
			}
			int offset = visualLines.First<VisualLine>().FirstDocumentLine.Offset;
			int endOffset = visualLines.Last<VisualLine>().LastDocumentLine.EndOffset;
			BackgroundGeometryBuilder backgroundGeometryBuilder = new BackgroundGeometryBuilder();
			backgroundGeometryBuilder.AlignToMiddleOfPixels = true;
			backgroundGeometryBuilder.CornerRadius = 1.0;
			foreach (SearchResult searchResult in this.Results)
			{
				if (searchResult.StartOffset >= offset && searchResult.EndOffset <= endOffset)
				{
					backgroundGeometryBuilder.AddSegment(textView, searchResult);
					Geometry geometry = backgroundGeometryBuilder.CreateGeometry();
					if (geometry != null)
					{
						drawingContext.DrawGeometry(SearchResultBackgroundRenderer._markerBrush, null, geometry);
					}
				}
			}
		}

		// Token: 0x0400003B RID: 59
		private static readonly Brush _markerBrush = SearchResultBackgroundRenderer.CreateBackgroundBrush();
	}
}
