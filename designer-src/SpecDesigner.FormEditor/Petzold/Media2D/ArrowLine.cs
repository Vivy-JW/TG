using System;
using System.Windows;
using System.Windows.Media;

namespace SpecDesigner.FormEditor.Petzold.Media2D
{
	// Token: 0x0200002F RID: 47
	public class ArrowLine : ArrowLineBase
	{
		// Token: 0x1700005A RID: 90
		// (get) Token: 0x060001AB RID: 427 RVA: 0x00008BCF File Offset: 0x00006DCF
		// (set) Token: 0x060001AA RID: 426 RVA: 0x00008BBC File Offset: 0x00006DBC
		public double X1
		{
			get
			{
				return (double)base.GetValue(ArrowLine.X1Property);
			}
			set
			{
				base.SetValue(ArrowLine.X1Property, value);
			}
		}

		// Token: 0x1700005B RID: 91
		// (get) Token: 0x060001AD RID: 429 RVA: 0x00008BF4 File Offset: 0x00006DF4
		// (set) Token: 0x060001AC RID: 428 RVA: 0x00008BE1 File Offset: 0x00006DE1
		public double Y1
		{
			get
			{
				return (double)base.GetValue(ArrowLine.Y1Property);
			}
			set
			{
				base.SetValue(ArrowLine.Y1Property, value);
			}
		}

		// Token: 0x1700005C RID: 92
		// (get) Token: 0x060001AF RID: 431 RVA: 0x00008C19 File Offset: 0x00006E19
		// (set) Token: 0x060001AE RID: 430 RVA: 0x00008C06 File Offset: 0x00006E06
		public double X2
		{
			get
			{
				return (double)base.GetValue(ArrowLine.X2Property);
			}
			set
			{
				base.SetValue(ArrowLine.X2Property, value);
			}
		}

		// Token: 0x1700005D RID: 93
		// (get) Token: 0x060001B1 RID: 433 RVA: 0x00008C3E File Offset: 0x00006E3E
		// (set) Token: 0x060001B0 RID: 432 RVA: 0x00008C2B File Offset: 0x00006E2B
		public double Y2
		{
			get
			{
				return (double)base.GetValue(ArrowLine.Y2Property);
			}
			set
			{
				base.SetValue(ArrowLine.Y2Property, value);
			}
		}

		// Token: 0x1700005E RID: 94
		// (get) Token: 0x060001B2 RID: 434 RVA: 0x00008C50 File Offset: 0x00006E50
		protected override Geometry DefiningGeometry
		{
			get
			{
				this.pathgeo.Figures.Clear();
				this.pathfigLine.StartPoint = new Point(this.X1, this.Y1);
				this.polysegLine.Points.Clear();
				this.polysegLine.Points.Add(new Point(this.X2, this.Y2));
				this.pathgeo.Figures.Add(this.pathfigLine);
				return base.DefiningGeometry;
			}
		}

		// Token: 0x040000F3 RID: 243
		public static readonly DependencyProperty X1Property = DependencyProperty.Register("X1", typeof(double), typeof(ArrowLine), new FrameworkPropertyMetadata(0.0, FrameworkPropertyMetadataOptions.AffectsMeasure));

		// Token: 0x040000F4 RID: 244
		public static readonly DependencyProperty Y1Property = DependencyProperty.Register("Y1", typeof(double), typeof(ArrowLine), new FrameworkPropertyMetadata(0.0, FrameworkPropertyMetadataOptions.AffectsMeasure));

		// Token: 0x040000F5 RID: 245
		public static readonly DependencyProperty X2Property = DependencyProperty.Register("X2", typeof(double), typeof(ArrowLine), new FrameworkPropertyMetadata(0.0, FrameworkPropertyMetadataOptions.AffectsMeasure));

		// Token: 0x040000F6 RID: 246
		public static readonly DependencyProperty Y2Property = DependencyProperty.Register("Y2", typeof(double), typeof(ArrowLine), new FrameworkPropertyMetadata(0.0, FrameworkPropertyMetadataOptions.AffectsMeasure));
	}
}
