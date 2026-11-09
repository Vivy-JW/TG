using System;
using System.Windows;
using System.Windows.Media;
using System.Windows.Shapes;

namespace SpecDesigner.FormEditor.Petzold.Media2D
{
	// Token: 0x0200001D RID: 29
	public abstract class ArrowLineBase : Shape
	{
		// Token: 0x17000031 RID: 49
		// (get) Token: 0x060000EC RID: 236 RVA: 0x0000563E File Offset: 0x0000383E
		// (set) Token: 0x060000EB RID: 235 RVA: 0x0000562B File Offset: 0x0000382B
		public double ArrowAngle
		{
			get
			{
				return (double)base.GetValue(ArrowLineBase.ArrowAngleProperty);
			}
			set
			{
				base.SetValue(ArrowLineBase.ArrowAngleProperty, value);
			}
		}

		// Token: 0x17000032 RID: 50
		// (get) Token: 0x060000EE RID: 238 RVA: 0x00005663 File Offset: 0x00003863
		// (set) Token: 0x060000ED RID: 237 RVA: 0x00005650 File Offset: 0x00003850
		public double ArrowLength
		{
			get
			{
				return (double)base.GetValue(ArrowLineBase.ArrowLengthProperty);
			}
			set
			{
				base.SetValue(ArrowLineBase.ArrowLengthProperty, value);
			}
		}

		// Token: 0x17000033 RID: 51
		// (get) Token: 0x060000F0 RID: 240 RVA: 0x00005688 File Offset: 0x00003888
		// (set) Token: 0x060000EF RID: 239 RVA: 0x00005675 File Offset: 0x00003875
		public ArrowEnds ArrowEnds
		{
			get
			{
				return (ArrowEnds)base.GetValue(ArrowLineBase.ArrowEndsProperty);
			}
			set
			{
				base.SetValue(ArrowLineBase.ArrowEndsProperty, value);
			}
		}

		// Token: 0x17000034 RID: 52
		// (get) Token: 0x060000F2 RID: 242 RVA: 0x000056AD File Offset: 0x000038AD
		// (set) Token: 0x060000F1 RID: 241 RVA: 0x0000569A File Offset: 0x0000389A
		public bool IsArrowClosed
		{
			get
			{
				return (bool)base.GetValue(ArrowLineBase.IsArrowClosedProperty);
			}
			set
			{
				base.SetValue(ArrowLineBase.IsArrowClosedProperty, value);
			}
		}

		// Token: 0x060000F3 RID: 243 RVA: 0x000056C0 File Offset: 0x000038C0
		public ArrowLineBase()
		{
			this.pathgeo = new PathGeometry();
			this.pathfigLine = new PathFigure();
			this.polysegLine = new PolyLineSegment();
			this.pathfigLine.Segments.Add(this.polysegLine);
			this.pathfigHead1 = new PathFigure();
			this.polysegHead1 = new PolyLineSegment();
			this.pathfigHead1.Segments.Add(this.polysegHead1);
			this.pathfigHead2 = new PathFigure();
			this.polysegHead2 = new PolyLineSegment();
			this.pathfigHead2.Segments.Add(this.polysegHead2);
		}

		// Token: 0x17000035 RID: 53
		// (get) Token: 0x060000F4 RID: 244 RVA: 0x00005764 File Offset: 0x00003964
		protected override Geometry DefiningGeometry
		{
			get
			{
				int count = this.polysegLine.Points.Count;
				if (count > 0)
				{
					if ((this.ArrowEnds & ArrowEnds.Start) == ArrowEnds.Start)
					{
						Point startPoint = this.pathfigLine.StartPoint;
						Point point = this.polysegLine.Points[0];
						this.pathgeo.Figures.Add(this.CalculateArrow(this.pathfigHead1, point, startPoint));
					}
					if ((this.ArrowEnds & ArrowEnds.End) == ArrowEnds.End)
					{
						Point point2 = ((count == 1) ? this.pathfigLine.StartPoint : this.polysegLine.Points[count - 2]);
						Point point3 = this.polysegLine.Points[count - 1];
						this.pathgeo.Figures.Add(this.CalculateArrow(this.pathfigHead2, point2, point3));
					}
				}
				return this.pathgeo;
			}
		}

		// Token: 0x060000F5 RID: 245 RVA: 0x0000583C File Offset: 0x00003A3C
		private PathFigure CalculateArrow(PathFigure pathfig, Point pt1, Point pt2)
		{
			Matrix matrix = default(Matrix);
			Vector vector = pt1 - pt2;
			vector.Normalize();
			vector *= this.ArrowLength;
			PolyLineSegment polyLineSegment = pathfig.Segments[0] as PolyLineSegment;
			polyLineSegment.Points.Clear();
			matrix.Rotate(this.ArrowAngle / 2.0);
			pathfig.StartPoint = pt2 + vector * matrix;
			polyLineSegment.Points.Add(pt2);
			matrix.Rotate(-this.ArrowAngle);
			polyLineSegment.Points.Add(pt2 + vector * matrix);
			pathfig.IsClosed = this.IsArrowClosed;
			if (this.IsArrowClosed)
			{
				base.Fill = base.Stroke;
			}
			return pathfig;
		}

		// Token: 0x04000088 RID: 136
		protected PathGeometry pathgeo;

		// Token: 0x04000089 RID: 137
		protected PathFigure pathfigLine;

		// Token: 0x0400008A RID: 138
		protected PolyLineSegment polysegLine;

		// Token: 0x0400008B RID: 139
		private PathFigure pathfigHead1;

		// Token: 0x0400008C RID: 140
		private PolyLineSegment polysegHead1;

		// Token: 0x0400008D RID: 141
		private PathFigure pathfigHead2;

		// Token: 0x0400008E RID: 142
		private PolyLineSegment polysegHead2;

		// Token: 0x0400008F RID: 143
		public static readonly DependencyProperty ArrowAngleProperty = DependencyProperty.Register("ArrowAngle", typeof(double), typeof(ArrowLineBase), new FrameworkPropertyMetadata(45.0, FrameworkPropertyMetadataOptions.AffectsMeasure));

		// Token: 0x04000090 RID: 144
		public static readonly DependencyProperty ArrowLengthProperty = DependencyProperty.Register("ArrowLength", typeof(double), typeof(ArrowLineBase), new FrameworkPropertyMetadata(12.0, FrameworkPropertyMetadataOptions.AffectsMeasure));

		// Token: 0x04000091 RID: 145
		public static readonly DependencyProperty ArrowEndsProperty = DependencyProperty.Register("ArrowEnds", typeof(ArrowEnds), typeof(ArrowLineBase), new FrameworkPropertyMetadata(ArrowEnds.End, FrameworkPropertyMetadataOptions.AffectsMeasure));

		// Token: 0x04000092 RID: 146
		public static readonly DependencyProperty IsArrowClosedProperty = DependencyProperty.Register("IsArrowClosed", typeof(bool), typeof(ArrowLineBase), new FrameworkPropertyMetadata(false, FrameworkPropertyMetadataOptions.AffectsMeasure));
	}
}
